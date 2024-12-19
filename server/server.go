package server

import (
	"context"
	"fmt"
	"time"

	pb "muzz-explore/internal/api"
	"muzz-explore/internal/store"

	"github.com/rs/zerolog/log"
)

func ref[T any](t T) *T { return &t }

//go:generate go run github.com/vektra/mockery/v2@v2.50.0 --with-expecter --name DecisionStore
type DecisionStore interface {
	ListDecisions(ctx context.Context, filter store.DecisionFilter, page string) ([]store.Decision, string, error)
	CountDecisions(ctx context.Context, filter store.DecisionFilter) (uint64, error)
	UpsertDecision(ctx context.Context, decision store.Decision) error
	MarkDecisionsAsSeen(ctx context.Context, RecipientUserID string, timestamp int64) error
}

type ServiceServer struct {
	pb.UnimplementedExploreServiceServer
	ds    DecisionStore
	nowFn func() time.Time // Used to get the current time, overridden in tests.
}

func NewServiceServer(ds DecisionStore) *ServiceServer {
	return &ServiceServer{
		ds:    ds,
		nowFn: time.Now,
	}
}

func (s *ServiceServer) ListLikedYou(
	ctx context.Context,
	in *pb.ListLikedYouRequest,
) (*pb.ListLikedYouResponse, error) {
	now := s.nowFn().Unix()
	decisions, nestPage, err := s.ds.ListDecisions(
		ctx,
		store.DecisionFilter{
			RecipientUserID: ref(in.GetRecipientUserId()),
			LikedRecipient:  ref(true),
		},
		in.GetPaginationToken(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list decisions: %v", err)
	}

	// Asynchronously update decisions sent to mark them as seen.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.ds.MarkDecisionsAsSeen(ctx, in.RecipientUserId, now); err != nil {
			log.Warn().Err(err).Msg("failed to mark decisions as seen")
		}
	}()

	return &pb.ListLikedYouResponse{
		Likers:              storeToListLikedYouResponse_Liker(decisions),
		NextPaginationToken: &nestPage,
	}, nil
}

func (s *ServiceServer) ListNewLikedYou(
	ctx context.Context,
	in *pb.ListLikedYouRequest,
) (*pb.ListLikedYouResponse, error) {
	now := s.nowFn().Unix()
	recipient := in.GetRecipientUserId()
	decisions, nextPage, err := s.ds.ListDecisions(
		ctx,
		store.DecisionFilter{
			RecipientUserID: &recipient,
			LikedRecipient:  ref(true),
			SeenByRecipient: ref(false),
		},
		in.GetPaginationToken(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list decisions: %v", err)
	}

	// Asynchronously update decisions sent to mark them as seen.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.ds.MarkDecisionsAsSeen(ctx, recipient, now); err != nil {
			log.Warn().Err(err).Msg("failed to mark decisions as seen")
		}
	}()

	return &pb.ListLikedYouResponse{
		Likers:              storeToListLikedYouResponse_Liker(decisions),
		NextPaginationToken: &nextPage,
	}, nil
}

func (s *ServiceServer) CountLikedYou(
	ctx context.Context,
	in *pb.CountLikedYouRequest,
) (*pb.CountLikedYouResponse, error) {
	count, err := s.ds.CountDecisions(ctx, store.DecisionFilter{
		RecipientUserID: ref(in.GetRecipientUserId()),
		LikedRecipient:  ref(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count decisions: %v", err)
	}
	return &pb.CountLikedYouResponse{Count: count}, nil
}

func (s *ServiceServer) PutDecision(
	ctx context.Context,
	in *pb.PutDecisionRequest,
) (*pb.PutDecisionResponse, error) {
	err := s.ds.UpsertDecision(ctx, store.Decision{
		ActorUserID:     in.GetActorUserId(),
		RecipientUserID: in.GetRecipientUserId(),
		LikedRecipient:  in.GetLikedRecipient(),
		LastModified:    s.nowFn().Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert decision: %v", err)
	}

	// Check if it's mutual.
	resp := pb.PutDecisionResponse{MutualLikes: false}
	if in.GetLikedRecipient() {
		decisions, _, err := s.ds.ListDecisions(ctx, store.DecisionFilter{
			ActorUserID:     ref(in.GetRecipientUserId()),
			RecipientUserID: ref(in.GetActorUserId()),
			LikedRecipient:  ref(true),
		}, "")
		if err != nil {
			return &resp, fmt.Errorf("failed to check if it's mutual: %v", err)
		}
		if len(decisions) > 0 {
			resp.MutualLikes = true
		}
	}
	return &resp, nil
}

func storeToListLikedYouResponse_Liker(decisions []store.Decision) []*pb.ListLikedYouResponse_Liker {
	var likers []*pb.ListLikedYouResponse_Liker
	for _, decision := range decisions {
		likers = append(likers, &pb.ListLikedYouResponse_Liker{
			ActorId:       decision.ActorUserID,
			UnixTimestamp: uint64(decision.LastModified),
		})
	}
	return likers
}
