import json
import unittest
from unittest.mock import MagicMock, patch

from producer import main as producer_main
from tracker import main as tracker_main

class TestProducer(unittest.TestCase):

    @patch('producer.Producer')
    def test_producer_integration(self, mock_producer_class):
        """Teste l'intégration du producteur Kafka en simulant l'envoi d'un message.

        Ce test vérifie que le producteur est correctement instancié, que la méthode `produce`
        est appelée avec les bons arguments (topic et message), et que la méthode `flush`
        est bien appelée pour garantir l'envoi du message.

        Args:
            mock_producer_class (MagicMock): Le mock de la classe `Producer` de Kafka.
        """
        # Crée une instance simulée du producteur
        mock_producer_instance = MagicMock()
        mock_producer_class.return_value = mock_producer_instance

        # Exécute la fonction principale du producteur
        producer_main()

        # Vérifie que le producteur a été initialisé avec la bonne configuration
        mock_producer_class.assert_called_once_with({"bootstrap.servers": "localhost:9092"})

        # Vérifie que la méthode `produce` a été appelée
        self.assertTrue(mock_producer_instance.produce.called)
        
        # Récupère les arguments de l'appel à `produce`
        args, kwargs = mock_producer_instance.produce.call_args
        
        # Vérifie que le topic est correct
        self.assertEqual(kwargs.get('topic'), 'orders')
        
        # Vérifie que la valeur (le message) est un JSON valide
        try:
            message_value = json.loads(kwargs.get('value').decode('utf-8'))
            self.assertIn('order_id', message_value)
            self.assertIn('user', message_value)
            self.assertIn('item', message_value)
            self.assertIn('quantity', message_value)
        except (json.JSONDecodeError, AttributeError):
            self.fail("Le message produit n'est pas un JSON valide ou n'est pas correctement formaté.")

        # Vérifie que la méthode `flush` a été appelée pour envoyer le message
        mock_producer_instance.flush.assert_called_once()

from tracker import main as tracker_main

class TestTracker(unittest.TestCase):

    @patch('tracker.Consumer')
    def test_consumer_initialization(self, mock_consumer_class):
        """Vérifie que le consommateur Kafka est initialisé et s'abonne correctement.

        Ce test s'assure que le consommateur est configuré avec les bons paramètres
        et qu'il s'abonne au topic 'orders'. Il simule une interruption pour
        éviter une boucle infinie.

        Args:
            mock_consumer_class (MagicMock): Le mock de la classe `Consumer` de Kafka.
        """
        mock_consumer_instance = MagicMock()
        # Simule une interruption immédiate pour sortir de la boucle while
        mock_consumer_instance.poll.side_effect = KeyboardInterrupt
        mock_consumer_class.return_value = mock_consumer_instance

        # Exécute la fonction principale du tracker
        tracker_main()

        # Vérifie que le consommateur a été initialisé avec la bonne configuration
        mock_consumer_class.assert_called_once_with({
            "bootstrap.servers": "localhost:9092",
            "group.id": "order-tracker",
            "auto.offset.reset": "earliest"
        })

        # Vérifie que le consommateur s'est abonné au bon topic
        mock_consumer_instance.subscribe.assert_called_once_with(['orders'])

        # Vérifie que la méthode `close` a été appelée
        mock_consumer_instance.close.assert_called_once()

    @patch('tracker.Consumer')
    @patch('builtins.print')
    def test_message_processing(self, mock_print, mock_consumer_class):
        """Teste le traitement d'un message reçu par le consommateur.

        Ce test simule la réception d'un message Kafka et vérifie que le message
        est correctement décodé, désérialisé et que les informations pertinentes
        sont affichées.

        Args:
            mock_print (MagicMock): Le mock de la fonction `print`.
            mock_consumer_class (MagicMock): Le mock de la classe `Consumer` de Kafka.
        """
        mock_consumer_instance = MagicMock()
        mock_message = MagicMock()

        # Configure le message simulé
        order_data = {'quantity': 5, 'item': 'coffee', 'user': 'john'}
        mock_message.value.return_value = json.dumps(order_data).encode('utf-8')
        mock_message.error.return_value = None

        # Le poll retourne le message une fois, puis une interruption
        mock_consumer_instance.poll.side_effect = [mock_message, KeyboardInterrupt]
        mock_consumer_class.return_value = mock_consumer_instance

        # Exécute la fonction principale du tracker
        tracker_main()

        # Vérifie que le message a été traité et que la sortie est correcte
        mock_print.assert_any_call("📦 Received order: 5 x coffee from john")

if __name__ == '__main__':
    unittest.main()
