package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

// battleSize — число медиа в одном батле (ТЗ 8.1.1).
const battleSize = 8

// StartBattle — ТЗ 8.1.1: случайно выбирает 8 медиа трипа и создаёт сессию батла.
// Клиент проводит турнир 4→2→1 локально и присылает финального победителя в SubmitBattleResult.
func (s *TripService) StartBattle(ctx context.Context, req *pb.StartBattleRequest) (*pb.StartBattleResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	total, _, err := s.mediaRepo.CountByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to count media")
	}
	// ТЗ 8.1.9: при < 8 медиа фотобатл недоступен.
	if total < battleSize {
		return nil, status.Errorf(codes.FailedPrecondition, "need at least %d media to start a battle", battleSize)
	}
	picked, err := s.mediaRepo.PickRandomForBattle(tripID, battleSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to pick media")
	}
	// Фильтр Restricted мог уронить выборку ниже порога: в таком случае пользовательских
	// доступных медиа реально < 8 — трактуем как то же условие 8.1.9.
	if len(picked) < battleSize {
		return nil, status.Errorf(codes.FailedPrecondition, "need at least %d available media to start a battle", battleSize)
	}
	mediaIDs := make([]string, 0, len(picked))
	for _, m := range picked {
		mediaIDs = append(mediaIDs, m.ID)
	}
	battle := &models.MediaBattle{TripID: tripID, UserID: userID, MediaIDs: mediaIDs}
	if err := s.battleRepo.Create(battle); err != nil {
		return nil, status.Error(codes.Internal, "failed to create battle")
	}
	outMedia := make([]*pb.BattleMedia, 0, len(picked))
	for _, m := range picked {
		outMedia = append(outMedia, &pb.BattleMedia{
			MediaId:   m.ID,
			Url:       s.presignedReadURL(ctx, m.S3Key),
			MediaType: m.MediaType,
		})
	}
	return &pb.StartBattleResponse{BattleId: battle.ID, Media: outMedia}, nil
}

// SubmitBattleResult — ТЗ 8.1.7-8.1.8: фиксирует финального победителя и инкрементит его battle_rating.
// Идемпотентная защита: если батл уже завершён — возвращает FailedPrecondition, повторный +1 невозможен.
func (s *TripService) SubmitBattleResult(ctx context.Context, req *pb.SubmitBattleResultRequest) (*pb.SubmitBattleResultResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	battleID := req.GetBattleId()
	winnerID := req.GetWinnerMediaId()
	if battleID == "" || winnerID == "" {
		return nil, status.Error(codes.InvalidArgument, "battle_id and winner_media_id are required")
	}
	battle, err := s.battleRepo.GetByID(battleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "battle not found")
		}
		return nil, status.Error(codes.Internal, "failed to get battle")
	}
	if battle.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "battle belongs to another user")
	}
	if battle.FinishedAt != nil {
		return nil, status.Error(codes.FailedPrecondition, "battle already finished")
	}
	if !slices.Contains(battle.MediaIDs, winnerID) {
		return nil, status.Error(codes.InvalidArgument, "winner_media_id is not part of this battle")
	}
	// Сначала закрываем сессию: SetWinner атомарно переводит finished_at с NULL на NOW() и защищает от гонок
	// (два параллельных SubmitBattleResult не смогут оба получить RowsAffected>0 — второй вернёт ErrNoRows).
	if err := s.battleRepo.SetWinner(battleID, winnerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.FailedPrecondition, "battle already finished")
		}
		return nil, status.Error(codes.Internal, "failed to finalize battle")
	}
	rating, err := s.mediaRepo.IncrementBattleRating(winnerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update battle rating")
	}
	return &pb.SubmitBattleResultResponse{NewBattleRating: rating}, nil
}

// GetBestMemories — ТЗ 8.2: медиа трипа с battle_rating > 0, отсортированные по убыванию рейтинга (story-mode).
// Если в трипе нет таких медиа — возвращается пустой массив; решение показывать режим принимает клиент (ТЗ 8.2.3).
func (s *TripService) GetBestMemories(ctx context.Context, req *pb.GetBestMemoriesRequest) (*pb.GetBestMemoriesResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	list, err := s.mediaRepo.ListWithPositiveBattleRating(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list best memories")
	}
	out := make([]*pb.BestMemory, 0, len(list))
	for _, m := range list {
		var capturedAtUnix int64
		if m.CapturedAt != nil {
			capturedAtUnix = m.CapturedAt.Unix()
		}
		out = append(out, &pb.BestMemory{
			MediaId:        m.ID,
			Url:            s.presignedReadURL(ctx, m.S3Key),
			MediaType:      m.MediaType,
			BattleRating:   m.BattleRating,
			CapturedAtUnix: capturedAtUnix,
		})
	}
	return &pb.GetBestMemoriesResponse{Media: out}, nil
}
