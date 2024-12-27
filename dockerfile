FROM python:3.12-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code to the container
COPY . .

# Expose the port
EXPOSE 8000

RUN chmod +x postgrest.sh 

# Command to run the application
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
#CMD ["./postgrest.sh"]
# postgrest.sh is a debugging script that checks db connection.
# It's saved because it's useful for debugging purposes.