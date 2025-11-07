# Démonstration d'un Pipeline Kafka

Ce projet a pour but de fournir une démonstration pratique d'un pipeline Kafka simple, orchestré avec Docker Compose, et comprenant un producteur ainsi qu'un consommateur développés en Python.

## Structure du Projet

- `producer.py` : Un script Python qui simule la création d'une commande et l'envoie sous forme de message au topic Kafka `orders`.
- `tracker.py` : Un script Python qui s'abonne au topic `orders`, écoute les messages entrants et les affiche en temps réel dans la console.
- `docker-compose.yaml` : Un fichier de configuration Docker Compose qui provisionne et lance un broker Kafka dans un conteneur.
- `requirements.txt` : La liste des dépendances Python nécessaires au fonctionnement du producteur et du consommateur.
- `tests/test_smoke.py` : Une suite de tests d'intégration de base qui s'assure que les modules `producer` et `tracker` peuvent être importés sans erreur.

## Prérequis

- **Docker et Docker Compose** : Pour exécuter l'environnement conteneurisé.
- **Python 3.9 ou supérieur** : Pour exécuter les scripts du producteur et du consommateur. `pip` est également requis pour l'installation des dépendances, et il est généralement inclus dans les installations de Python.

## Installation

1. **Démarrer le broker Kafka :**

   ```bash
   docker-compose up -d
   ```

2. **Installer les dépendances Python :**

   ```bash
   pip install -r requirements.txt
   ```

## Utilisation

1. **Lancer le consommateur :**

   Ouvrez un terminal et exécutez la commande suivante pour démarrer le consommateur. Le consommateur attendra des messages sur le topic `orders`.

   ```bash
   python3 tracker.py
   ```

2. **Lancer le producteur :**

   Ouvrez un autre terminal et exécutez la commande suivante pour envoyer un exemple de message au topic `orders`.

   ```bash
   python3 producer.py
   ```

   Vous devriez voir le message apparaître dans le terminal du consommateur.

## Développement

Ce projet utilise `pytest` pour les tests et `flake8` pour le linting.

- **Exécuter les tests :**

  ```bash
  PYTHONPATH=. python3 -m pytest
  ```

- **Exécuter le linting :**

  ```bash
  flake8 .
  ```

## Documentation

Le code est entièrement documenté en suivant les conventions de style Google pour les docstrings Python. Chaque fonction publique, méthode et classe possède une docstring détaillée qui explique son but, ses paramètres et sa valeur de retour.

## Commandes Kafka

Voici quelques commandes utiles pour interagir avec Kafka :

- **Lister tous les topics :**

  ```bash
  docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
  ```

- **Décrire un topic :**

  ```bash
  docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
  ```

- **Voir tous les événements dans un topic :**

  ```bash
  docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
  ```
