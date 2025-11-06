# Cours accéléré sur Kafka

Ce projet présente une configuration Kafka simple utilisant Docker Compose, avec un producteur et un consommateur Python.

## Structure du projet

- `producer.py` : Un script Python qui envoie un exemple de message au topic Kafka `orders`.
- `tracker.py` : Un script Python qui s'abonne au topic `orders` et affiche les messages reçus.
- `docker-compose.yaml` : Un fichier de configuration Docker Compose pour démarrer un broker Kafka.
- `requirements.txt` : Une liste des dépendances Python requises pour ce projet.
- `.github/workflows/ci.yml` : Un pipeline d'intégration continue qui exécute les tests et le linting à chaque push ou pull request.
- `tests/test_smoke.py` : Une suite de tests de base pour s'assurer que les modules `producer` et `tracker` peuvent être importés.

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
  pytest
  ```

- **Exécuter le linting :**

  ```bash
  flake8 .
  ```

## Intégration continue (CI/CD)

Ce projet utilise GitHub Actions pour l'intégration continue. Le pipeline est défini dans le fichier `.github/workflows/ci.yml` et exécute les étapes suivantes :

- Installe les dépendances Python.
- Exécute le linting avec `flake8`.
- Exécute les tests avec `pytest`.

Le pipeline est déclenché à chaque `push` et `pull request` sur la branche `main`.

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
