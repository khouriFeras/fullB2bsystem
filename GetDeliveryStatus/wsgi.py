"""
WSGI entry point for Gunicorn (production).
Usage: gunicorn --bind 0.0.0.0:${PORT:-5000} --workers 2 --threads 2 --timeout 60 wsgi:app
"""
from app import app
