from __future__ import annotations

import uuid

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.engine import build_sampling_params, get_engine

router = APIRouter(prefix="/inference", tags=["inference"])

# LLaMA Guard 2 conversation template
_LLAMA_GUARD_SYSTEM = (
    "<|begin_of_text|><|start_header_id|>user<|end_header_id|>\n\n"
    "Task: Check if there is unsafe content in the following message according "
    "to our safety policy.\n\n<BEGIN CONVERSATION>\nUser: {message}\n<END CONVERSATION>\n\n"
    "Provide your safety assessment for the message above:\n"
    "- First line must read 'safe' or 'unsafe'.\n"
    "- If unsafe, a second line must include a comma-separated list of violated categories."
    "<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n"
)


class InferenceRequest(BaseModel):
    message: str
    request_id: str | None = None


class InferenceResponse(BaseModel):
    request_id: str
    output: str
    flagged: bool
    violated_categories: list[str]


@router.post("/llama-guard", response_model=InferenceResponse)
async def run_llama_guard(body: InferenceRequest) -> InferenceResponse:
    if not body.message.strip():
        raise HTTPException(status_code=422, detail="message must not be empty")

    request_id = body.request_id or str(uuid.uuid4())
    prompt = _LLAMA_GUARD_SYSTEM.format(message=body.message)

    engine = get_engine()
    sampling = build_sampling_params()

    output_text = ""
    async for result in engine.generate(prompt, sampling, request_id=request_id):
        if result.outputs:
            output_text = result.outputs[0].text

    lines = output_text.strip().splitlines()
    flagged = lines[0].strip().lower() == "unsafe" if lines else False
    categories = [c.strip() for c in lines[1].split(",")] if flagged and len(lines) > 1 else []

    return InferenceResponse(
        request_id=request_id,
        output=output_text,
        flagged=flagged,
        violated_categories=categories,
    )
