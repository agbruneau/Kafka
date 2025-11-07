def test_import_producer():
    """Vérifie que le module `producer` peut être importé sans erreur.

    Ce test a pour but de s'assurer que le module `producer` et toutes ses
    dépendances sont correctement installés et accessibles. Une `ImportError`
    indiquerait un problème dans l'environnement ou dans la structure du projet.
    """
    try:
        import producer
    except ImportError:
        assert False, "Failed to import producer module"

def test_import_tracker():
    """Vérifie que le module `tracker` peut être importé sans erreur.

    Ce test a pour but de s'assurer que le module `tracker` et toutes ses
    dépendances sont correctement installés et accessibles. Une `ImportError`
    indiquerait un problème dans l'environnement ou dans la structure du projet.
    """
    try:
        import tracker
    except ImportError:
        assert False, "Failed to import tracker module"
