package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/notification-service/internal/mocks"
	"pinz/backend/notification-service/internal/models"
	pb "pinz/backend/notification-service/pkg/proto"
)

func TestDispatchTrips_SendsToEnabledParticipantsOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tripCli := mocks.NewMockTripClientInterface(ctrl)
	tokens := mocks.NewMockDeviceTokensRepositoryInterface(ctrl)
	notifLog := mocks.NewMockNotificationLogRepositoryInterface(ctrl)
	apnsMock := mocks.NewMockSender(ctrl)

	trip := &pb.NotificationTrip{TripId: "t1", ParticipantUserIds: []string{"u1", "u2"}}

	tripCli.EXPECT().GetNotificationSettings(gomock.Any(), "t1", []string{"u1", "u2"}).
		Return(map[string]bool{"u1": true, "u2": false}, nil)

	tokens.EXPECT().ListByUsers(gomock.Any(), []string{"u1"}).
		Return([]models.DeviceToken{{APNSToken: "tok-a"}, {APNSToken: "tok-b"}}, nil)

	notifLog.EXPECT().IsSent(gomock.Any(), gomock.Any(), "tok-a").Return(false, nil)
	notifLog.EXPECT().IsSent(gomock.Any(), gomock.Any(), "tok-b").Return(true, nil)
	apnsMock.EXPECT().Send(gomock.Any(), "tok-a", gomock.Any()).Return(nil)
	notifLog.EXPECT().MarkSent(gomock.Any(), gomock.Any(), "tok-a").Return(true, nil)

	d := Deps{Tokens: tokens, NotifLog: notifLog, TripClient: tripCli, APNS: apnsMock}
	dispatchTrips(context.Background(), d, []*pb.NotificationTrip{trip}, "TRIP_ANNIVERSARY", "Pinz", "body")
}

func TestDispatchTrips_SkipsEmptyRecipients(t *testing.T) {
	d := Deps{}
	dispatchTrips(context.Background(), d, []*pb.NotificationTrip{{TripId: "t", ParticipantUserIds: nil}}, "X", "T", "B")
	// Никаких вызовов к mock-ам → тест пройден если не было паники.
	require.True(t, true)
}
