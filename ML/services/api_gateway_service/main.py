from fastapi import FastAPI, HTTPException
import requests

app = FastAPI(title='API Gateway (HTTP -> services)')

MODERATION_URL = 'http://localhost:8001/moderate'
TRIP_URL = 'http://localhost:8002/classify'

@app.post('/moderate')
async def moderate(payload: dict):
    # For production, implement proper gRPC client using generated stubs from proto
    try:
        r = requests.post(MODERATION_URL, json=payload, timeout=5)
        return r.json()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post('/classify')
async def classify(payload: dict):
    try:
        r = requests.post(TRIP_URL, json=payload, timeout=5)
        return r.json()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get('/health')
async def health():
    return {'status': 'ok'}
