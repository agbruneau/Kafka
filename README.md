# Cours accéléré sur Kafka

Ce projet est une démonstration simple de l'utilisation de Kafka avec un producteur et un consommateur Python.

## Prérequis

- Docker et Docker Compose
- Python 3
- `pip` pour l'installation des dépendances Python

## Installation

1. Clonez ce dépôt.
2. Installez les dépendances Python nécessaires :

   ```bash
   pip install -r requirements.txt
   ```

## Utilisation

Pour lancer l'environnement de test, exécutez le script `start.sh` :

```bash
./start.sh
```

Ce script effectuera les actions suivantes :

1.  Démarrage d'un conteneur Kafka avec Docker Compose.
2.  Attente de 30 secondes pour garantir que Kafka est pleinement opérationnel.
3.  Lancement du consommateur (`tracker.py`) en arrière-plan. Les journaux de sortie seront enregistrés dans `tracker.log`.
4.  Exécution du producteur (`producer.py`) pour envoyer un message de test.

Pour arrêter l'environnement de test et nettoyer, exécutez le script `stop.sh` :

```bash
./stop.sh
```

Ce script arrêtera le conteneur Kafka et le processus du consommateur.

## Commandes utiles

Quelques commandes utiles pour interagir avec Kafka directement :

- **Lister les topics Kafka :**

  ```bash
  docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
  ```

- **Décrire un topic pour voir ses partitions :**

  ```bash
  docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
  ```

- **Afficher tous les événements d'un topic :**

  ```bash
  docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
  ```
