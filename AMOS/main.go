package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/config"
	"github.com/HarshD0011/AMOS/AMOS/k8s"
	"github.com/HarshD0011/AMOS/AMOS/notification"
	"github.com/HarshD0011/AMOS/AMOS/service"
	"github.com/HarshD0011/AMOS/AMOS/tools"
)

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
	debug      = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	// 1. Load Configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("Warning: Could not load config file: %v. Using environment variables/defaults.", err)
		cfg = config.LoadFromEnv()
	}

	// Validate critical config
	if cfg.ADK.APIKey == "" {
		// Log warning only for now to allow partial startup if user just wants monitoring but no AI
		log.Println("WARNING: GOOGLE_API_KEY is not set. Remediation agent will fail.")
	}

	if *debug {
		log.Println("Debug mode enabled")
	}

	// 2. Setup Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Handle Signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Initiating shutdown...", sig)
		cancel()
	}()

	log.Println("AMOS: Autonomous Multi-agent Orchestration Service starting...")

	// 4. Initialize Core Components
	
	// K8s Client
	k8sClient, err := k8s.NewClient(&cfg.Kubernetes)
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}
	log.Println("Connected to Kubernetes cluster")

	// 5. Initialize Services
	
	// Channels
	faultOrchChan := make(chan service.UnifiedFault, 100)
	
	// Fault Detector
	faultDetector := service.NewFaultDetector(faultOrchChan)
	podFaultChan, deployFaultChan, jobFaultChan := faultDetector.GetChannels()

	// Monitors
	podMonitor := k8s.NewPodMonitor(k8sClient, podFaultChan)
	deployMonitor := k8s.NewDeploymentMonitor(k8sClient, deployFaultChan)
	jobMonitor := k8s.NewJobMonitor(k8sClient, jobFaultChan)

	// Utils
	k8sTools := tools.NewK8sTools(k8sClient)
	ctxGen := service.NewContextGenerator(k8sTools)
	snapshotSvc := service.NewSnapshotService(k8sClient)
	retryMgr := service.NewRetryManager(cfg.Remediation)
	rollbackSvc := service.NewRollbackService(k8sClient, snapshotSvc)
	
	// Notification
	emailSvc := notification.NewEmailService(cfg.Email)
	escSvc := notification.NewEscalationService(emailSvc, cfg.Email.EngineerEmail)

	// Agent
	remediationAgent, err := agent.NewRemediationAgent(&cfg.ADK, k8sTools)
	if err != nil {
		log.Fatalf("Failed to create Remediation Agent: %v", err)
	}

	// Orchestrator
	orchestrator := service.NewOrchestrator(
		faultDetector,
		remediationAgent,
		ctxGen,
		snapshotSvc,
		retryMgr,
		rollbackSvc,
		escSvc,
		faultOrchChan,
	)

	// 6. Start Background Routines
	go podMonitor.Start(ctx)
	go deployMonitor.Start(ctx)
	go jobMonitor.Start(ctx)
	go faultDetector.Start(ctx)
	go orchestrator.Start(ctx)

	log.Println("AMOS is fully operational and watching for faults.")

	// 7. Wait
	<-ctx.Done()
	log.Println("Shutdown complete.")
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	return config.LoadFromFile(path)
}