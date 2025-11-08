# Cours accéléré sur Kafka avec Python

Ce projet est une démonstration simple de l'utilisation d'Apache Kafka avec un producteur et un consommateur développés en Python. L'objectif est de fournir un exemple clair et concis pour les débutants qui souhaitent comprendre les concepts de base de Kafka.

## Table des matières

- [Présentation du projet](#présentation-du-projet)
- [Structure du projet](#structure-du-projet)
- [Prérequis](#prérequis)
- [Installation](#installation)
- [Utilisation](#utilisation)
- [Commandes utiles](#commandes-utiles)

## Présentation du projet

Ce projet simule un système de commande simple où :
- Un **producteur** (`producer.py`) envoie des messages de commande à un topic Kafka.
- Un **consommateur** (`tracker.py`) s'abonne à ce topic pour recevoir et afficher les messages de commande en temps réel.

L'ensemble de l'environnement Kafka est géré par Docker Compose, ce qui simplifie grandement le déploiement et la gestion des services.

## Structure du projet

```
.
├── docker-compose.yaml   # Fichier de configuration pour l'environnement Docker
├── producer.py           # Script du producteur Kafka
├── README.md             # Ce fichier
├── requirements.txt      # Dépendances Python
├── start.sh              # Script pour démarrer l'environnement
├── stop.sh               # Script pour arrêter l'environnement
└── tracker.py            # Script du consommateur Kafka
```

## Prérequis

- Docker et Docker Compose
- Python 3
- `pip` pour l'installation des dépendances Python

## Installation

1.  **Clonez ce dépôt :**
    ```bash
    git clone https://github.com/votre-utilisateur/votre-projet.git
    cd votre-projet
    ```

2.  **Installez les dépendances Python :**
    ```bash
    pip install -r requirements.txt
    ```

## Utilisation

Pour lancer l'environnement de test, exécutez le script `start.sh` :

```bash
./start.sh
```

Ce script effectuera les actions suivantes :

1.  Démarrage d'un conteneur Kafka avec Docker Compose.
2.  Attente de 30 secondes pour garantir que Kafka est pleinement opérationnel.
3.  Lancement du consommateur (`tracker.py`) en arrière-plan. Les journaux de sortie seront enregistrés dans `tracker.log`.
4.  Exécution du producteur (`producer.py`) pour envoyer un message de test.

Pour arrêter l'environnement de test et nettoyer, exécutez le script `stop.sh` :

```bash
./stop.sh
```

Ce script arrêtera le conteneur Kafka et le processus du consommateur.

## Commandes utiles

Quelques commandes utiles pour interagir avec Kafka directement :

- **Lister les topics Kafka :**
  ```bash
  docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
  ```

- **Décrire un topic pour voir ses partitions :**
  ```bash
  docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
  ```

- **Afficher tous les événements d'un topic :**
  ```bash
  docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
  ```
