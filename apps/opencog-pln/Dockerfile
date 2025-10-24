FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY app.py .

# Expose port
EXPOSE 8102

# Set environment variables
ENV PORT=8102
ENV HOST=0.0.0.0
ENV PYTHONUNBUFFERED=1

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8102/healthcheck')"

# Run with gunicorn for production
CMD ["gunicorn", "--bind", "0.0.0.0:8102", "--workers", "4", "--timeout", "120", "app:app"]
