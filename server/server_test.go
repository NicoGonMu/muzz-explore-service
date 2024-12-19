package server

import (
	"context"
	"fmt"
	pb "muzz-explore/internal/api"
	"muzz-explore/internal/store"
	"muzz-explore/server/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListLikedYou(t *testing.T) {
	testMap := map[string]struct {
		storeReturnedDecisions []store.Decision
		storeReturnedPageToken string
		storeReturnedError     error
		in                     *pb.ListLikedYouRequest
		wantErr                error
		want                   *pb.ListLikedYouResponse
	}{
		"no decisions": {
			in:                     &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: nil,
			storeReturnedError:     nil,
			wantErr:                nil,
			want:                   &pb.ListLikedYouResponse{Likers: nil},
		},
		"multiple decisions": {
			in: &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: []store.Decision{
				{ActorUserID: "user2", RecipientUserID: "user1", LikedRecipient: true, LastModified: 1},
				{ActorUserID: "user3", RecipientUserID: "user1", LikedRecipient: false, LastModified: 2},
				{ActorUserID: "user4", RecipientUserID: "user1", LikedRecipient: false, LastModified: 3},
			},
			storeReturnedPageToken: "user4##user1",
			storeReturnedError:     nil,
			wantErr:                nil,
			want: &pb.ListLikedYouResponse{
				Likers: []*pb.ListLikedYouResponse_Liker{
					{ActorId: "user2", UnixTimestamp: 1},
					{ActorId: "user3", UnixTimestamp: 2},
					{ActorId: "user4", UnixTimestamp: 3},
				},
				NextPaginationToken: ref("user4##user1"),
			},
		},
		"error listing decisions": {
			in:                     &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: nil,
			storeReturnedError:     fmt.Errorf("some error"),
			wantErr:                fmt.Errorf("failed to list decisions: some error"),
			want:                   nil,
		},
	}
	for name, tc := range testMap {
		t.Run(name, func(t *testing.T) {
			// GIVEN: A Server with some preconditions.
			ctx := context.Background()
			dsMock := mocks.NewDecisionStore(t)
			inPaginationToken := tc.in.GetPaginationToken()
			dsMock.EXPECT().ListDecisions(
				ctx,
				store.DecisionFilter{
					RecipientUserID: ref("user1"),
					LikedRecipient:  ref(true),
				},
				inPaginationToken,
			).Return(tc.storeReturnedDecisions, tc.storeReturnedPageToken, tc.storeReturnedError)
			if tc.storeReturnedError == nil && tc.want.GetNextPaginationToken() != "" {
				// As error is only logged, we don't care about the return value.
				dsMock.EXPECT().MarkDecisionsAsSeen(
					mock.Anything, // context internally created
					"user1",
					inPaginationToken,
					tc.want.GetNextPaginationToken(),
				).Return(nil)
			}
			s := NewServiceServer(dsMock)

			// WHEN: ListLikedYou is called.
			got, err := s.ListLikedYou(ctx, tc.in)
			time.Sleep(10 * time.Millisecond) // Wait for the goroutine to finish.

			// THEN: The result should match the expectations.
			require.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListNewLikedYou(t *testing.T) {
	testMap := map[string]struct {
		storeReturnedDecisions []store.Decision
		storeReturnedPageToken string
		storeReturnedError     error
		in                     *pb.ListLikedYouRequest
		wantErr                error
		want                   *pb.ListLikedYouResponse
	}{
		"no decisions": {
			in:                     &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: nil,
			storeReturnedError:     nil,
			wantErr:                nil,
			want:                   &pb.ListLikedYouResponse{Likers: nil},
		},
		"multiple decisions": {
			in: &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: []store.Decision{
				{ActorUserID: "user2", RecipientUserID: "user1", LikedRecipient: true, LastModified: 1},
				{ActorUserID: "user3", RecipientUserID: "user1", LikedRecipient: false, LastModified: 2},
				{ActorUserID: "user4", RecipientUserID: "user1", LikedRecipient: false, LastModified: 3},
			},
			storeReturnedPageToken: "user4##user1",
			storeReturnedError:     nil,
			wantErr:                nil,
			want: &pb.ListLikedYouResponse{
				Likers: []*pb.ListLikedYouResponse_Liker{
					{ActorId: "user2", UnixTimestamp: 1},
					{ActorId: "user3", UnixTimestamp: 2},
					{ActorId: "user4", UnixTimestamp: 3},
				},
				NextPaginationToken: ref("user4##user1"),
			},
		},
		"error listing decisions": {
			in:                     &pb.ListLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedDecisions: nil,
			storeReturnedError:     fmt.Errorf("some error"),
			wantErr:                fmt.Errorf("failed to list decisions: some error"),
			want:                   nil,
		},
	}
	for name, tc := range testMap {
		t.Run(name, func(t *testing.T) {
			// GIVEN: A Server with some preconditions.
			ctx := context.Background()
			inPaginationToken := tc.in.GetPaginationToken()
			dsMock := mocks.NewDecisionStore(t)
			dsMock.EXPECT().ListDecisions(
				ctx,
				store.DecisionFilter{
					RecipientUserID: ref("user1"),
					LikedRecipient:  ref(true),
					SeenByRecipient: ref(false),
				},
				inPaginationToken,
			).Return(tc.storeReturnedDecisions, tc.storeReturnedPageToken, tc.storeReturnedError)
			if tc.storeReturnedError == nil && tc.want.GetNextPaginationToken() != "" {
				// As error is only logged, we don't care about the return value.
				dsMock.EXPECT().MarkDecisionsAsSeen(
					mock.Anything, // context internally created
					"user1",
					inPaginationToken,
					tc.want.GetNextPaginationToken(),
				).Return(nil)
			}

			s := NewServiceServer(dsMock)

			// WHEN: ListNewLikedYou is called.
			got, err := s.ListNewLikedYou(ctx, tc.in)
			time.Sleep(10 * time.Millisecond) // Wait for the goroutine to finish.

			// THEN: The result should match the expectations.
			require.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountLikedYou(t *testing.T) {
	testMap := map[string]struct {
		storeReturnedCount       uint64
		storeReturnedError       error
		dacisionStoreMockFactory func(context.Context) DecisionStore
		in                       *pb.CountLikedYouRequest
		wantErr                  error
		want                     *pb.CountLikedYouResponse
	}{
		"no decisions": {
			in:                 &pb.CountLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedCount: 0,
			storeReturnedError: nil,
			wantErr:            nil,
			want:               &pb.CountLikedYouResponse{Count: 0},
		},
		"multiple likes": {
			in:                 &pb.CountLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedCount: 3,
			storeReturnedError: nil,
			wantErr:            nil,
			want:               &pb.CountLikedYouResponse{Count: 3},
		},
		"error counting decisions": {
			in:                 &pb.CountLikedYouRequest{RecipientUserId: "user1"},
			storeReturnedCount: 0,
			storeReturnedError: fmt.Errorf("some error"),
			wantErr:            fmt.Errorf("failed to count decisions: some error"),
			want:               nil,
		},
	}
	for name, tc := range testMap {
		t.Run(name, func(t *testing.T) {
			// GIVEN: A Server with some preconditions.
			ctx := context.Background()
			dsMock := mocks.NewDecisionStore(t)
			dsMock.EXPECT().CountDecisions(ctx, store.DecisionFilter{
				RecipientUserID: ref("user1"),
				LikedRecipient:  ref(true),
			}).Return(tc.storeReturnedCount, tc.storeReturnedError)
			s := NewServiceServer(dsMock)

			// WHEN: CountLikedYou is called.
			got, err := s.CountLikedYou(ctx, tc.in)

			// THEN: The result should match the expectations.
			require.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPutDecision(t *testing.T) {
	testMap := map[string]struct {
		in                       *pb.PutDecisionRequest
		decisionStoreMockFactory func(ctx context.Context, lastModified int64) DecisionStore
		wantErr                  error
		want                     *pb.PutDecisionResponse
	}{
		"pass decision": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  false,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  false,
					LastModified:    lastModified,
				}).Return(nil)
				return dsMock
			},
			wantErr: nil,
			want:    &pb.PutDecisionResponse{MutualLikes: false},
		},
		"like decision, no reverse decision": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  true,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  true,
					LastModified:    lastModified,
				}).Return(nil)
				dsMock.EXPECT().ListDecisions(ctx, store.DecisionFilter{
					ActorUserID:     ref("user1"),
					RecipientUserID: ref("user2"),
					LikedRecipient:  ref(true),
				}, "").Return(nil, "", nil)
				return dsMock
			},
			wantErr: nil,
			want:    &pb.PutDecisionResponse{MutualLikes: false},
		},
		"like decision, it's a match": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  true,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  true,
					LastModified:    lastModified,
				}).Return(nil)
				dsMock.EXPECT().ListDecisions(ctx, store.DecisionFilter{
					ActorUserID:     ref("user1"),
					RecipientUserID: ref("user2"),
					LikedRecipient:  ref(true),
				}, "").Return([]store.Decision{
					{ActorUserID: "user1", RecipientUserID: "user2", LikedRecipient: true, LastModified: 1},
				}, "user2##user1", nil)
				return dsMock
			},
			wantErr: nil,
			want:    &pb.PutDecisionResponse{MutualLikes: true},
		},
		"like decision, it's not a match": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  true,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  true,
					LastModified:    lastModified,
				}).Return(nil)
				dsMock.EXPECT().ListDecisions(ctx, store.DecisionFilter{
					ActorUserID:     ref("user1"),
					RecipientUserID: ref("user2"),
					LikedRecipient:  ref(true),
				}, "").Return(nil, "", nil)
				return dsMock
			},
			wantErr: nil,
			want:    &pb.PutDecisionResponse{MutualLikes: false},
		},
		"error putting decision": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  false,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  false,
					LastModified:    lastModified,
				}).Return(fmt.Errorf("some error"))
				return dsMock
			},
			wantErr: fmt.Errorf("failed to upsert decision: some error"),
			want:    nil,
		},
		"error checking it's a match": {
			in: &pb.PutDecisionRequest{
				RecipientUserId: "user1",
				ActorUserId:     "user2",
				LikedRecipient:  true,
			},
			decisionStoreMockFactory: func(ctx context.Context, lastModified int64) DecisionStore {
				dsMock := mocks.NewDecisionStore(t)
				dsMock.EXPECT().UpsertDecision(ctx, store.Decision{
					RecipientUserID: "user1",
					ActorUserID:     "user2",
					LikedRecipient:  true,
					LastModified:    lastModified,
				}).Return(nil)
				dsMock.EXPECT().ListDecisions(ctx, store.DecisionFilter{
					ActorUserID:     ref("user1"),
					RecipientUserID: ref("user2"),
					LikedRecipient:  ref(true),
				}, "").Return(nil, "", fmt.Errorf("some error"))
				return dsMock
			},
			wantErr: fmt.Errorf("failed to check if it's mutual: some error"),
			want:    &pb.PutDecisionResponse{MutualLikes: false},
		},
	}
	for name, tc := range testMap {
		t.Run(name, func(t *testing.T) {
			// GIVEN: A Server with some preconditions.
			ctx := context.Background()
			now := time.Now()
			s := NewServiceServer(tc.decisionStoreMockFactory(ctx, now.Unix()))
			s.nowFn = func() time.Time { return now }

			// WHEN: ListLikedYou is called.
			got, err := s.PutDecision(ctx, tc.in)

			// THEN: The result should match the expectations.
			require.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
