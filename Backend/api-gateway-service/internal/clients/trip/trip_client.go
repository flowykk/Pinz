package trip

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"pinz/backend/api-gateway-service/internal/middleware"
	pb "pinz/backend/api-gateway-service/pkg/proto"
)

const metadataUserIDKey = "x-user-id"

type Client struct {
	conn *grpc.ClientConn
	client pb.TripServiceClient
}

func NewClient() (*Client, error) {
	addr := os.Getenv("TRIP_SERVICE_GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:50052"
		slog.Warn("TRIP_SERVICE_GRPC_ADDRESS not set, using localhost:50052")
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("trip gRPC client: %w", err)
	}
	return &Client{
		conn: conn,
		client: pb.NewTripServiceClient(conn),
	}, nil
}

// withUserIDMetadata adds x-user-id from context to outgoing gRPC metadata.
func withUserIDMetadata(ctx context.Context) context.Context {
	userID := middleware.UserIDFromContext(ctx)
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, metadataUserIDKey, userID)
}

func (c *Client) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	return c.client.CreateTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	return c.client.GetTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) ListUserTrips(ctx context.Context, req *pb.ListUserTripsRequest) (*pb.ListUserTripsResponse, error) {
	return c.client.ListUserTrips(withUserIDMetadata(ctx), req)
}

func (c *Client) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	return c.client.UpdateTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) DeleteTrip(ctx context.Context, req *pb.DeleteTripRequest) (*pb.DeleteTripResponse, error) {
	return c.client.DeleteTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) RequestTripCoverUpload(ctx context.Context, req *pb.RequestTripCoverUploadRequest) (*pb.RequestTripCoverUploadResponse, error) {
	return c.client.RequestTripCoverUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) ConfirmTripCoverUpload(ctx context.Context, req *pb.ConfirmTripCoverUploadRequest) (*pb.ConfirmTripCoverUploadResponse, error) {
	return c.client.ConfirmTripCoverUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) DeleteTripCover(ctx context.Context, req *pb.DeleteTripCoverRequest) (*pb.DeleteTripCoverResponse, error) {
	return c.client.DeleteTripCover(withUserIDMetadata(ctx), req)
}

func (c *Client) GenerateInviteLink(ctx context.Context, req *pb.GenerateInviteLinkRequest) (*pb.GenerateInviteLinkResponse, error) {
	return c.client.GenerateInviteLink(withUserIDMetadata(ctx), req)
}

func (c *Client) JoinTripByToken(ctx context.Context, req *pb.JoinTripByTokenRequest) (*pb.JoinTripByTokenResponse, error) {
	return c.client.JoinTripByToken(withUserIDMetadata(ctx), req)
}

func (c *Client) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*pb.RemoveParticipantResponse, error) {
	return c.client.RemoveParticipant(withUserIDMetadata(ctx), req)
}

func (c *Client) LeaveTrip(ctx context.Context, req *pb.LeaveTripRequest) (*pb.LeaveTripResponse, error) {
	return c.client.LeaveTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) ProcessMediaGrouping(ctx context.Context, req *pb.ProcessMediaGroupingRequest) (*pb.ProcessMediaGroupingResponse, error) {
	return c.client.ProcessMediaGrouping(withUserIDMetadata(ctx), req)
}

func (c *Client) ApplyGroupsAndProcess(ctx context.Context, req *pb.ApplyGroupsAndProcessRequest) (*pb.ApplyGroupsAndProcessResponse, error) {
	return c.client.ApplyGroupsAndProcess(withUserIDMetadata(ctx), req)
}

func (c *Client) GetTripReview(ctx context.Context, req *pb.GetTripReviewRequest) (*pb.GetTripReviewResponse, error) {
	return c.client.GetTripReview(withUserIDMetadata(ctx), req)
}

func (c *Client) FinalizeTrip(ctx context.Context, req *pb.FinalizeTripRequest) (*pb.FinalizeTripResponse, error) {
	return c.client.FinalizeTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) PublishTrip(ctx context.Context, req *pb.PublishTripRequest) (*pb.PublishTripResponse, error) {
	return c.client.PublishTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) UpdateTripSettings(ctx context.Context, req *pb.UpdateTripSettingsRequest) (*pb.UpdateTripSettingsResponse, error) {
	return c.client.UpdateTripSettings(withUserIDMetadata(ctx), req)
}

func (c *Client) ListFeed(ctx context.Context, req *pb.ListFeedRequest) (*pb.ListFeedResponse, error) {
	return c.client.ListFeed(withUserIDMetadata(ctx), req)
}

func (c *Client) LikeTrip(ctx context.Context, req *pb.LikeTripRequest) (*pb.LikeTripResponse, error) {
	return c.client.LikeTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) DislikeTrip(ctx context.Context, req *pb.DislikeTripRequest) (*pb.DislikeTripResponse, error) {
	return c.client.DislikeTrip(withUserIDMetadata(ctx), req)
}

func (c *Client) AddToFavourites(ctx context.Context, req *pb.AddToFavouritesRequest) (*pb.AddToFavouritesResponse, error) {
	return c.client.AddToFavourites(withUserIDMetadata(ctx), req)
}

func (c *Client) RemoveFromFavourites(ctx context.Context, req *pb.RemoveFromFavouritesRequest) (*pb.RemoveFromFavouritesResponse, error) {
	return c.client.RemoveFromFavourites(withUserIDMetadata(ctx), req)
}

func (c *Client) ListFavourites(ctx context.Context, req *pb.ListFavouritesRequest) (*pb.ListFavouritesResponse, error) {
	return c.client.ListFavourites(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaStart(ctx context.Context, req *pb.AddMediaStartRequest) (*pb.AddMediaStartResponse, error) {
	return c.client.AddMediaStart(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaRequestUploadUrls(ctx context.Context, req *pb.AddMediaRequestUploadUrlsRequest) (*pb.AddMediaRequestUploadUrlsResponse, error) {
	return c.client.AddMediaRequestUploadUrls(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaCommitUpload(ctx context.Context, req *pb.AddMediaCommitUploadRequest) (*pb.AddMediaCommitUploadResponse, error) {
	return c.client.AddMediaCommitUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaGetSessionMedia(ctx context.Context, req *pb.AddMediaGetSessionMediaRequest) (*pb.AddMediaGetSessionMediaResponse, error) {
	return c.client.AddMediaGetSessionMedia(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaProcessGrouping(ctx context.Context, req *pb.AddMediaProcessGroupingRequest) (*pb.AddMediaProcessGroupingResponse, error) {
	return c.client.AddMediaProcessGrouping(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaGetGrouping(ctx context.Context, req *pb.AddMediaGetGroupingRequest) (*pb.AddMediaGetGroupingResponse, error) {
	return c.client.AddMediaGetGrouping(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaApplyGroupsAndProcess(ctx context.Context, req *pb.AddMediaApplyGroupsAndProcessRequest) (*pb.AddMediaApplyGroupsAndProcessResponse, error) {
	return c.client.AddMediaApplyGroupsAndProcess(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaGetReview(ctx context.Context, req *pb.AddMediaGetReviewRequest) (*pb.AddMediaGetReviewResponse, error) {
	return c.client.AddMediaGetReview(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaConfirm(ctx context.Context, req *pb.AddMediaConfirmRequest) (*pb.AddMediaConfirmResponse, error) {
	return c.client.AddMediaConfirm(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaCancel(ctx context.Context, req *pb.AddMediaCancelRequest) (*pb.AddMediaCancelResponse, error) {
	return c.client.AddMediaCancel(withUserIDMetadata(ctx), req)
}

func (c *Client) AddMediaTakeover(ctx context.Context, req *pb.AddMediaTakeoverRequest) (*pb.AddMediaTakeoverResponse, error) {
	return c.client.AddMediaTakeover(withUserIDMetadata(ctx), req)
}

func (c *Client) ListUserTripSummaries(ctx context.Context, req *pb.ListUserTripSummariesRequest) (*pb.ListUserTripSummariesResponse, error) {
	return c.client.ListUserTripSummaries(withUserIDMetadata(ctx), req)
}

func (c *Client) StartBattle(ctx context.Context, req *pb.StartBattleRequest) (*pb.StartBattleResponse, error) {
	return c.client.StartBattle(withUserIDMetadata(ctx), req)
}

func (c *Client) SubmitBattleResult(ctx context.Context, req *pb.SubmitBattleResultRequest) (*pb.SubmitBattleResultResponse, error) {
	return c.client.SubmitBattleResult(withUserIDMetadata(ctx), req)
}

func (c *Client) GetBestMemories(ctx context.Context, req *pb.GetBestMemoriesRequest) (*pb.GetBestMemoriesResponse, error) {
	return c.client.GetBestMemories(withUserIDMetadata(ctx), req)
}

func (c *Client) SearchPins(ctx context.Context, req *pb.SearchPinsRequest) (*pb.SearchPinsResponse, error) {
	return c.client.SearchPins(withUserIDMetadata(ctx), req)
}

func (c *Client) UpsertTripPrivacy(ctx context.Context, req *pb.UpsertTripPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	return c.client.UpsertTripPrivacy(withUserIDMetadata(ctx), req)
}

func (c *Client) UpsertPinPrivacy(ctx context.Context, req *pb.UpsertPinPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	return c.client.UpsertPinPrivacy(withUserIDMetadata(ctx), req)
}

func (c *Client) UpsertMediaPrivacy(ctx context.Context, req *pb.UpsertMediaPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	return c.client.UpsertMediaPrivacy(withUserIDMetadata(ctx), req)
}

// Pin RUD

func (c *Client) GetPin(ctx context.Context, req *pb.GetPinRequest) (*pb.GetPinResponse, error) {
	return c.client.GetPin(withUserIDMetadata(ctx), req)
}

func (c *Client) UpdatePin(ctx context.Context, req *pb.UpdatePinRequest) (*pb.UpdatePinResponse, error) {
	return c.client.UpdatePin(withUserIDMetadata(ctx), req)
}

func (c *Client) DeletePin(ctx context.Context, req *pb.DeletePinRequest) (*pb.DeletePinResponse, error) {
	return c.client.DeletePin(withUserIDMetadata(ctx), req)
}

func (c *Client) RemoveMediaFromPin(ctx context.Context, req *pb.RemoveMediaFromPinRequest) (*pb.RemoveMediaFromPinResponse, error) {
	return c.client.RemoveMediaFromPin(withUserIDMetadata(ctx), req)
}

// Pin upload flow (унифицированный creation/addition).

func (c *Client) PinUploadStart(ctx context.Context, req *pb.PinUploadStartRequest) (*pb.PinUploadStartResponse, error) {
	return c.client.PinUploadStart(withUserIDMetadata(ctx), req)
}

func (c *Client) RequestPinUploadUrls(ctx context.Context, req *pb.RequestPinUploadUrlsRequest) (*pb.RequestPinUploadUrlsResponse, error) {
	return c.client.RequestPinUploadUrls(withUserIDMetadata(ctx), req)
}

func (c *Client) CommitPinUpload(ctx context.Context, req *pb.CommitPinUploadRequest) (*pb.CommitPinUploadResponse, error) {
	return c.client.CommitPinUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) ProcessPinUpload(ctx context.Context, req *pb.ProcessPinUploadRequest) (*pb.ProcessPinUploadResponse, error) {
	return c.client.ProcessPinUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) GetPinUploadReview(ctx context.Context, req *pb.GetPinUploadReviewRequest) (*pb.GetPinUploadReviewResponse, error) {
	return c.client.GetPinUploadReview(withUserIDMetadata(ctx), req)
}

func (c *Client) FinalizePinUpload(ctx context.Context, req *pb.FinalizePinUploadRequest) (*pb.FinalizePinUploadResponse, error) {
	return c.client.FinalizePinUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) CancelPinUpload(ctx context.Context, req *pb.CancelPinUploadRequest) (*pb.CancelPinUploadResponse, error) {
	return c.client.CancelPinUpload(withUserIDMetadata(ctx), req)
}

func (c *Client) GetRecommendations(ctx context.Context, req *pb.GetRecommendationsRequest) (*pb.GetRecommendationsResponse, error) {
	return c.client.GetRecommendations(withUserIDMetadata(ctx), req)
}

func (c *Client) SaveRecommendation(ctx context.Context, req *pb.SaveRecommendationRequest) (*pb.SaveRecommendationResponse, error) {
	return c.client.SaveRecommendation(withUserIDMetadata(ctx), req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
