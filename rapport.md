### Rapport Final

Ce rapport détaille le fonctionnement d'un projet de démonstration Apache Kafka utilisant Python et Docker. Il est divisé en trois sections :
1.  **Analyse technique** : Une explication du fonctionnement de chaque composant du projet.
2.  **Vulgarisation** : Une analogie simple pour comprendre les concepts de Kafka.
3.  **Validation pratique** : Les résultats de la tentative d'exécution du projet et les problèmes rencontrés.

---

### 1. Analyse Technique

Le projet est une application simple mais bien structurée qui démontre le modèle de messagerie *producteur-consommateur* avec Kafka.

*   **`docker-compose.yaml`** : Ce fichier définit l'infrastructure. Il met en place un service unique nommé `kafka` en utilisant l'image officielle de Confluent. Ce service expose le port `9092`, qui est le port par défaut pour les clients Kafka. La configuration utilise le mode Kraft, une version moderne de Kafka qui ne nécessite pas Zookeeper, ce qui simplifie le déploiement.

*   **`producer.py`** : Ce script Python joue le rôle du **producteur**. Sa tâche est de créer des messages et de les envoyer à Kafka.
    *   Il se connecte au broker Kafka à l'adresse `localhost:9092`.
    *   Il génère des messages de commande (au format JSON) contenant un ID de commande unique, un nom d'utilisateur, un article et une quantité.
    *   Il envoie ces messages au *topic* (sujet) `orders` toutes les deux secondes.
    *   Il inclut une fonction de rappel (`delivery_report`) pour confirmer que chaque message a bien été livré au broker.

*   **`tracker.py`** : Ce script est le **consommateur**. Il écoute les messages envoyés à un topic Kafka.
    *   Il se connecte au même broker Kafka et s'abonne au topic `orders`.
    *   Il fait partie d'un `group.id` nommé `order-tracker`, ce qui signifie que plusieurs instances de ce consommateur pourraient se répartir la charge si nécessaire.
    *   Grâce à `auto.offset.reset: "earliest"`, s'il est lancé, il lira tous les messages du topic depuis le tout début, même ceux envoyés avant son démarrage.
    *   Lorsqu'il reçoit un message, il le décode et affiche son contenu dans la console.

*   **`start.sh` et `stop.sh`** : Ces scripts shell automatisent la gestion de l'environnement.
    *   `start.sh` : Démarre le conteneur Kafka, attend 30 secondes pour s'assurer qu'il est pleinement opérationnel, installe les dépendances Python, lance le consommateur `tracker.py` en arrière-plan (en enregistrant sa sortie dans `tracker.log`), puis exécute le producteur `producer.py` au premier plan.
    *   `stop.sh` : Arrête proprement le conteneur Kafka et termine les processus du producteur et du consommateur.

---

### 2. Vulgarisation : Kafka comme un Bureau de Poste

Pour comprendre Kafka, imaginons un **bureau de poste géant et très efficace**.

*   **Le Broker Kafka (le Bureau de Poste)** : C'est le centre de tri. Il ne fait que recevoir et distribuer du courrier, sans se soucier de son contenu.

*   **Les Topics (les Guichets)** : Imaginez des guichets spécifiques pour différents types de courrier. Dans ce projet, il y a un guichet unique nommé `orders`. Tous les courriers liés aux commandes passent par ce guichet.

*   **Le Producteur (`producer.py`) (L'Expéditeur)** : C'est une personne (`lara` dans le script) qui écrit des lettres (les messages de commande) et les dépose continuellement au guichet `orders`. Chaque lettre a un contenu unique (un ID de commande différent).

*   **Le Consommateur (`tracker.py`) (Le Destinataire)** : C'est une autre personne qui attend au guichet `orders`. Dès qu'une nouvelle lettre arrive, il la prend, la lit et annonce son contenu à voix haute. Il est très attentif et ne manque aucune lettre.

En résumé, le producteur **envoie** des messages à un sujet spécifique dans Kafka, et le consommateur **reçoit** ces messages en s'abonnant au même sujet. Kafka agit comme un intermédiaire fiable et rapide qui garantit que les messages ne sont pas perdus en cours de route.

---

### 3. Validation Pratique

Lors de la tentative d'exécution du projet via le script `./start.sh`, j'ai rencontré un problème qui a empêché le système de démarrer.

*   **Action** : J'ai lancé `./start.sh`.
*   **Observation** : Le script a tenté de démarrer le conteneur Docker, mais a échoué.
*   **Diagnostic** : En examinant les logs de Docker, j'ai identifié l'erreur suivante :
    ```
    toomanyrequests: You have reached your unauthenticated pull rate limit.
    ```
*   **Cause** : Cette erreur est due à une limitation de Docker Hub. Pour les utilisateurs non authentifiés, Docker impose une limite sur le nombre d'images qui peuvent être téléchargées sur une certaine période. L'environnement d'exécution a dépassé cette limite, et Docker a donc refusé de télécharger l'image `confluentinc/cp-kafka`.

*   **Conclusion de la validation** : Le projet lui-même est **théoriquement fonctionnel** et bien conçu. Cependant, son exécution dépend d'un service externe (Docker Hub) qui a des limitations. **Le projet ne fonctionnera pas pour un utilisateur qui a dépassé son quota de téléchargement d'images Docker anonyme.**

Pour corriger ce problème, un utilisateur devrait s'authentifier à Docker Hub avec `docker login` avant d'exécuter le script `start.sh`. Cette information pourrait être ajoutée à la section `Prérequis` du fichier `README.md`.
