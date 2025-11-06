def test_import_producer():
    try:
        import producer
    except ImportError:
        assert False, "Failed to import producer module"

def test_import_tracker():
    try:
        import tracker
    except ImportError:
        assert False, "Failed to import tracker module"
