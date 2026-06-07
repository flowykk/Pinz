from __future__ import annotations

import asyncio

import grpc

from pinz_ai_proto import classifier_pb2, classifier_pb2_grpc
from pinz_shared.grpc_runtime import serve_grpc
from pinz_shared.logging import configure_logging, get_logger

from .head import Classifier
from .settings import Settings


log = get_logger("classifier")


class ClassifierServicer(classifier_pb2_grpc.ClassifierServicer):
    def __init__(self, clf: Classifier, min_conf: float) -> None:
        self._clf = clf
        self._min_conf = min_conf

    async def ClassifyPin(
        self,
        request: classifier_pb2.ClassifyPinRequest,
        context: grpc.aio.ServicerContext,
    ) -> classifier_pb2.ClassifyPinResponse:
        vectors = [list(e.vector) for e in request.image_embeddings]
        category, conf = self._clf.predict(vectors)
        if conf < self._min_conf:
            log.info("low confidence, fallback to custom",
                     pin_id=request.pin_id, conf=conf, top=category)
            category = "custom"
        tags = [category] if category != "custom" else []
        return classifier_pb2.ClassifyPinResponse(
            pin_id=request.pin_id,
            category=category,
            tags=tags,
            confidence=conf,
        )


async def main() -> None:
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)

    clf = Classifier(
        emb_dim=settings.embedding_dim,
        categories=list(settings.categories),
        head_path=settings.head_weights,
        prototypes_path=settings.zeroshot_prototypes,
    )
    servicer = ClassifierServicer(clf, settings.min_confidence)

    async def register(server: grpc.aio.Server) -> None:
        classifier_pb2_grpc.add_ClassifierServicer_to_server(servicer, server)

    await serve_grpc(settings.grpc_port, register, service_name=settings.service_name)


if __name__ == "__main__":
    asyncio.run(main())
