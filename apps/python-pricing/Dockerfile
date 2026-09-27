FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py ./

EXPOSE 8082
# OTel is set up in app.py (explicit), so plain uvicorn is enough.
ENTRYPOINT ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8082"]
