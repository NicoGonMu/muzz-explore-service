package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "muzz-explore/internal/api"
	database "muzz-explore/internal/store/database"
	server "muzz-explore/server"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const configFilePath = "/etc/explore-svc/config.json"

// Configuration for the service.
type Configuration struct {
	DBUser string `json:"dbUser"`
	DBPass string `json:"dbPass"`
	DBHost string `json:"dbHost"`
	DBPort string `json:"dbPort"`
	DBName string `json:"dbName"`
}

func main() {
	cfg, err := readConfig(configFilePath)
	if err != nil {
		log.Fatal().Msgf("Error reading config file: %v", err)
	}

	db, dbClose, err := database.NewClient(cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	if err != nil {
		log.Fatal().Msgf("failed to create database client: %v", err)
	}
	explorerService := server.NewServiceServer(db)

	tcpListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal().Msgf("failed to listen on port 8080: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterExploreServiceServer(s, explorerService)
	log.Printf("Explorer service listening at %v", tcpListener.Addr())
	go func() {
		if err := s.Serve(tcpListener); err != nil {
			log.Fatal().Msgf("failed to serve: %v", err)
		}
	}()

	WaitForShutdown()
	log.Info().Msg("interruption signal received, gracefully shutting down")

	// Close server.
	s.Stop()

	// If ServiceServer needed closing, it would go here, before the DB. Another possibility is
	// relaying DB closing to the ServiceServer as that's the module using it.

	// Close DB.
	if err := dbClose(); err != nil {
		log.Warn().Err(err).Msg("failed to mark decisions as seen")
	}
}

func WaitForShutdown(closers ...io.Closer) {
	c := make(chan os.Signal, 1)
	signal.Notify(
		c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM,
	)
	<-c
}

func readConfig(path string) (*Configuration, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := configFile.Close(); err != nil {
			log.Warn().Msgf("failed to close config file: %v", err)
		}
	}()

	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Configuration{}
	if err := json.Unmarshal(byteValue, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return &cfg, nil
}
