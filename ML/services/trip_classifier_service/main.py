from fastapi import FastAPI
from pydantic import BaseModel
from typing import Optional

app = FastAPI(title='Trip Classifier Service')

class TripRequest(BaseModel):
    text: str

@app.post('/classify')
async def classify(req: TripRequest):
    # Placeholder simple rule-based classification
    text = req.text.lower()
    if 'ski' in text or 'mountain' in text:
        return {'trip_type': 'adventure'}
    if 'beach' in text or 'sun' in text:
        return {'trip_type': 'relaxation'}
    return {'trip_type': 'unknown'}

@app.get('/health')
async def health():
    return {'status': 'ok'}
