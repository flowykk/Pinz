"""gRPC stub-clients для downstream сервисов."""

from __future__ import annotations

import grpc

from pinz_ai_proto import (
    classifier_pb2,
    classifier_pb2_grpc,
    common_pb2,
    embedding_pb2,
    embedding_pb2_grpc,
    moderation_pb2,
    moderation_pb2_grpc,
    similarity_pb2,
    similarity_pb2_grpc,
)


CHANNEL_OPTS = [
    ("grpc.max_send_message_length", 64 * 1024 * 1024),
    ("grpc.max_receive_message_length", 64 * 1024 * 1024),
]


class GrpcClients:
    def __init__(
        self,
        moderation: str,
        embedding: str,
        classifier: str,
        similarity: str,
    ) -> None:
        self._chan_moderation = grpc.aio.insecure_channel(moderation, options=CHANNEL_OPTS)
        self._chan_embedding = grpc.aio.insecure_channel(embedding, options=CHANNEL_OPTS)
        self._chan_classifier = grpc.aio.insecure_channel(classifier, options=CHANNEL_OPTS)
        self._chan_similarity = grpc.aio.insecure_channel(similarity, options=CHANNEL_OPTS)

        self.moderation = moderation_pb2_grpc.ModerationStub(self._chan_moderation)
        self.embedding = embedding_pb2_grpc.EmbeddingStub(self._chan_embedding)
        self.classifier = classifier_pb2_grpc.ClassifierStub(self._chan_classifier)
        self.similarity = similarity_pb2_grpc.SimilarityStub(self._chan_similarity)

    async def close(self) -> None:
        for ch in (
            self._chan_moderation,
            self._chan_embedding,
            self._chan_classifier,
            self._chan_similarity,
        ):
            await ch.close()


__all__ = [
    "GrpcClients",
    "classifier_pb2",
    "common_pb2",
    "embedding_pb2",
    "moderation_pb2",
    "similarity_pb2",
]
