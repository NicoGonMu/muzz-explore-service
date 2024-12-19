package main

import (
	"context"
	"fmt"
	pb "muzz-explore/internal/api"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// This will be the function in charge of running the tests.
// It assumes the service and db are already running.
// The flow of the test is as follows:
// 1. Put 3 decisions into the service.
// 2. Count the decisions for each user.
// 3. List new likes.
// 4. List new likes again (responses should be empty).
// 5. Put new decision.
// 6. List new likes.
// 7. List all likes.
func main() {
	ctx := context.Background()
	con, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Msgf("failed to create client: %v", err)
	}
	pbcl := pb.NewExploreServiceClient(con)

	fmt.Println("Connection established: Running tests...")

	// 1. Put 3 decisions into the service.
	// Decision 1: User 1 doesn't like user 2.
	if err := putDecision(ctx, pbcl, "1", "2", false, false); err != nil {
		log.Fatal().Msgf("failed to put decision 1: %v", err)
	}
	// Decision 2: User 1 likes user 3.
	if err := putDecision(ctx, pbcl, "1", "3", true, false); err != nil {
		log.Fatal().Msgf("failed to put decision 2: %v", err)
	}
	// Decision 3: User 3 likes user 1.
	if err := putDecision(ctx, pbcl, "3", "1", true, true); err != nil {
		log.Fatal().Msgf("failed to put decision 3: %v", err)
	}

	fmt.Println("Decisions introduced")

	// 2. Count the decisions for each user.
	if err := countDecisions(ctx, pbcl); err != nil {
		log.Fatal().Msgf("failed to count decisions: %v", err)
	}

	fmt.Println("Decisions counted")
	time.Sleep(time.Second)

	// 3. List new likes.
	if err := listNewLikes(ctx, pbcl, "1", []string{"3"}); err != nil {
		log.Fatal().Msgf("failed to list new likes for user 1: %v", err)
	}
	if err := listNewLikes(ctx, pbcl, "3", []string{"1"}); err != nil {
		log.Fatal().Msgf("failed to list new likes for user 3: %v", err)
	}

	// 4. List new likes again (responses should be empty).
	if err := listNewLikes(ctx, pbcl, "1", nil); err != nil {
		log.Fatal().Msgf("failed to list new likes for user 1: %v", err)
	}
	fmt.Println("New likes listed")

	// 5. Put new decision.
	if err := putDecision(ctx, pbcl, "2", "1", true, false); err != nil {
		log.Fatal().Msgf("failed to put decisions: %v", err)
	}

	// 6. List new likes.
	if err := listNewLikes(ctx, pbcl, "1", []string{"2"}); err != nil {
		log.Fatal().Msgf("failed to list new likes for user 1: %v", err)
	}

	// 7. List all likes.
	if err := listLikes(ctx, pbcl, "1", []string{"2", "3"}); err != nil {
		log.Fatal().Msgf("failed to list likes for user 1: %v", err)
	}

	fmt.Println("All likes listed")
	fmt.Println("All the checks passed!")
}

func putDecision(
	ctx context.Context,
	pbcl pb.ExploreServiceClient,
	actorUser, recipientUser string,
	liked, shouldBeMutual bool,
) error {
	resp, err := pbcl.PutDecision(ctx, &pb.PutDecisionRequest{
		ActorUserId:     actorUser,
		RecipientUserId: recipientUser,
		LikedRecipient:  liked,
	})
	if err != nil {
		return fmt.Errorf("failed to put decision: %w", err)
	}
	if resp.GetMutualLikes() != shouldBeMutual {
		return fmt.Errorf("unexpected mutual likes: got %v, want %v", resp.GetMutualLikes(), shouldBeMutual)
	}
	return nil
}

func listNewLikes(
	ctx context.Context,
	pbcl pb.ExploreServiceClient,
	recipientUser string,
	wantDecisions []string,
) error {
	resp, err := pbcl.ListNewLikedYou(ctx, &pb.ListLikedYouRequest{RecipientUserId: recipientUser})
	if err != nil {
		return fmt.Errorf("failed to list new likes: %w", err)
	}
	if len(resp.GetLikers()) != len(wantDecisions) {
		return fmt.Errorf("unexpected number of new likes: got %d, want %d", len(resp.GetLikers()), len(wantDecisions))
	}
	errMsg := ""
	for i := range resp.GetLikers() {
		if resp.GetLikers()[i].GetActorId() != wantDecisions[i] {
			errMsg += fmt.Sprintf(
				"unexpected new like: got %s, want %s\n",
				resp.GetLikers()[i].GetActorId(),
				wantDecisions[i],
			)
		}
	}
	if errMsg != "" {
		return fmt.Errorf("error matching new likes: %s", errMsg)
	}
	return nil
}

func listLikes(
	ctx context.Context,
	pbcl pb.ExploreServiceClient,
	recipientUser string,
	wantDecisions []string,
) error {
	resp, err := pbcl.ListLikedYou(ctx, &pb.ListLikedYouRequest{RecipientUserId: recipientUser})
	if err != nil {
		return fmt.Errorf("failed to list likes: %w", err)
	}
	if len(resp.GetLikers()) != len(wantDecisions) {
		return fmt.Errorf("unexpected number of new likes: got %d, want %d", len(resp.GetLikers()), len(wantDecisions))
	}
	errMsg := ""
	for i := range resp.GetLikers() {
		if resp.GetLikers()[i].GetActorId() != wantDecisions[i] {
			errMsg += fmt.Sprintf(
				"unexpected new like: got %s, want %s\n",
				resp.GetLikers()[i].GetActorId(),
				wantDecisions[i],
			)
		}
	}
	if errMsg != "" {
		return fmt.Errorf("error matching new likes: %s", errMsg)
	}
	return nil
}

func countDecisions(ctx context.Context, pbcl pb.ExploreServiceClient) error {
	// Count likes for user 1.
	count, err := pbcl.CountLikedYou(ctx, &pb.CountLikedYouRequest{RecipientUserId: "1"})
	if err != nil {
		return fmt.Errorf("failed to count decisions for user 1: %v", err)
	}
	if count.GetCount() != 1 {
		return fmt.Errorf("unexpected count for user 1 (expected 1): %v", count.GetCount())
	}

	// Count likes for user 2.
	count, err = pbcl.CountLikedYou(ctx, &pb.CountLikedYouRequest{RecipientUserId: "2"})
	if err != nil {
		log.Fatal().Msgf("failed to count decisions for user 2: %v", err)
	}
	if count.Count != 0 {
		log.Fatal().Msgf("unexpected count for user 2 (expected 0): %v", count.GetCount())
	}

	// Count likes for user 3.
	count, err = pbcl.CountLikedYou(ctx, &pb.CountLikedYouRequest{RecipientUserId: "3"})
	if err != nil {
		log.Fatal().Msgf("failed to count decisions for user 3: %v", err)
	}
	if count.Count != 1 {
		log.Fatal().Msgf("unexpected count for user 3 (expected 1): %v", count.GetCount())
	}
	return nil
}
