package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
	"github.com/nats-io/stan.go"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	cacheData   = make(map[int]orderData, 256)
	databaseURI = "port=5433 host=localhost user=wb_admin password=wb_admin dbname=wb_l0 sslmode=disable"
	clusterID   = "simple-cluster"
	clientID    = "testClient"
	natsURI     = "nats://localhost:4222"
)

func main() {

	db, err := sql.Open("postgres", databaseURI)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	rows, err := db.Query(`SELECT * FROM orders`)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var orderID int

		var jsonStr string
		var order = orderData{}
		err = rows.Scan(&orderID, &jsonStr)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal([]byte(jsonStr), &order)
		if err != nil {
			log.Fatal(err)
		}
		cacheData[orderID] = order
	}

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURI))
	if err != nil {
		log.Fatalf("Не удалось подключиться к NATS Streaming: %v", err)
	}

	defer sc.Close()

	sub, err := sc.Subscribe("chan1", func(msg *stan.Msg) {

		data := orderData{}
		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Fatal(err)
		}

		stmt := `
				INSERT INTO orders(data)
				VALUES ($1)
				RETURNING id`

		var orderID int

		err = db.QueryRow(stmt, msg.Data).Scan(&orderID)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("Received data through", msg.Subject)
		cacheData[orderID] = data

	})
	if err != nil {
		log.Fatalf("Не удалось подписаться на канал: %v", err)
	}

	defer sub.Unsubscribe()

	time.Sleep(1 * time.Second)

	server := http.NewServeMux()
	server.HandleFunc("GET /order/{id}", getOrder)
	log.Fatal(http.ListenAndServe(":8080", server))
}

func getOrder(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "incorrect id", http.StatusBadRequest)
	}

	result, ok := cacheData[id]

	if ok {
		jsonData, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		w.Write(jsonData)
	} else {
		http.Error(w, "data not found", http.StatusNotFound)
	}
}
