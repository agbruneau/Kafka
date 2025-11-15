package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	// Configuration du producteur Kafka.
	// "bootstrap.servers" est l'adresse du (ou des) broker(s) Kafka.
	producerConfig := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	}

	// Crée une nouvelle instance du producteur.
	producer, err := kafka.NewProducer(&producerConfig)
	if err != nil {
		fmt.Printf("Erreur fatale lors de la création du producteur: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	// Crée un canal pour recevoir les rapports de livraison.
	// La goroutine deliveryReport écoutera sur ce canal.
	deliveryChan := make(chan kafka.Event, 10000)
	go deliveryReport(deliveryChan)

	topic := "orders"

	// --- Mode d'Exécution Spécial pour les Tests d'Intégration ---
	// Vérifie la présence d'une variable d'environnement pour activer un mode
	// où un seul message est envoyé. C'est une technique courante pour rendre
	// les applications testables en intégration sans modifier leur code principal de manière invasive.
	if os.Getenv("SINGLE_MESSAGE_MODE") == "true" {
		payload := os.Getenv("SINGLE_MESSAGE_PAYLOAD")
		if payload == "" {
			fmt.Println("Erreur: SINGLE_MESSAGE_PAYLOAD ne doit pas être vide en mode single message")
			os.Exit(1)
		}
		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(payload),
		}, deliveryChan)
		if err != nil {
			fmt.Printf("Erreur lors de la production du message de test: %v\n", err)
			os.Exit(1)
		}
		producer.Flush(15 * 1000) // Attendre la livraison
		fmt.Println("✅ Message de test unique envoyé avec succès.")
		return // Terminer le programme après l'envoi
	}

	// --- Exécution Normale ---
	fmt.Println("🟢 Le producteur est démarré et prêt à envoyer des messages...")
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Utilisation de templates pour générer des données de commande variées.
	orderTemplates := []OrderTemplate{
		{User: "client01", Item: "espresso", Quantity: 2, Price: 2.50},
		{User: "client02", Item: "cappuccino", Quantity: 3, Price: 3.20},
		{User: "client03", Item: "latte", Quantity: 4, Price: 3.50},
		{User: "client04", Item: "macchiato", Quantity: 5, Price: 3.00},
		{User: "client05", Item: "flat white", Quantity: 6, Price: 3.30},
		{User: "client06", Item: "mocha", Quantity: 7, Price: 4.00},
		{User: "client07", Item: "americano", Quantity: 8, Price: 2.80},
		{User: "client08", Item: "chai latte", Quantity: 9, Price: 3.80},
		{User: "client09", Item: "matcha", Quantity: 10, Price: 4.50},
		{User: "client10", Item: "smoothie fraise", Quantity: 11, Price: 5.50},
	}

	// Boucle principale de production de messages.
	sequence := 1
	run := true
	for run {
		select {
		case <-sigchan:
			// Si un signal d'arrêt est reçu, on sort de la boucle.
			fmt.Println("\n⚠️  Signal d'arrêt reçu. Fin de la production de nouveaux messages...")
			run = false
		default:
			// Étape 1: Créer une commande enrichie en utilisant la fonction dédiée.
			template := orderTemplates[sequence%len(orderTemplates)]
			order := createOrder(sequence, template)

			// Étape 2: Sérialiser l'objet Order en JSON.
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("Erreur de sérialisation JSON: %v\n", err)
				continue
			}

			// Envoi du message au topic Kafka de manière asynchrone.
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          value,
			}, deliveryChan)

			if err != nil {
				fmt.Printf("Erreur lors de la production du message: %v\n", err)
			}

			sequence++
			time.Sleep(2 * time.Second) // Pause entre les envois.
		}
	}

	// Avant de terminer, vider le tampon du producteur pour garantir que tous les
	// messages en attente sont envoyés. C'est une étape cruciale pour l'arrêt propre.
	fmt.Println("⏳ Envoi des messages restants en file d'attente...")
	remainingMessages := producer.Flush(15 * 1000) // Timeout de 15 secondes.
	if remainingMessages > 0 {
		fmt.Printf("⚠️  %d messages n'ont pas pu être envoyés.\n", remainingMessages)
	} else {
		fmt.Println("✅ Tous les messages ont été envoyés avec succès.")
	}
}
