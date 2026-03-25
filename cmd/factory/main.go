package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/factory"
	"github.com/uttufy/FactoryAI/internal/job"
	"github.com/uttufy/FactoryAI/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "factory",
	Short: "FactoryAI - Multi-agent workspace manager",
	Long:  "A manufacturing-plant-inspired multi-agent workspace manager in Go.",
}

var (
	// v0.x flags
	blueprintPath string
	task          string
	noTUI         bool

	// v1.0 flags
	configPath   string
	projectPath  string
	stationName  string
	batchName    string
	formulaPath  string
	priority     int
	message      string
	operatorID   string
	mrID         string
	reason       string
	stationID    string
	beadID       string
	sopID        string
	role         string
	maxStations  int
	fromStation  string
	toStation    string
	workOnTraveler bool
)

func init() {
	// v0.x commands
	initV0Commands()

	// v1.0 commands
	initV1Commands()
}

func initV0Commands() {
	blueprintsDir := "./blueprints"

	runCmd := &cobra.Command{
		Use:   "run --blueprint <path> --task <task>",
		Short: "Run a factory blueprint",
		Long:  "Execute a factory blueprint with the given task.",
		Args:  cobra.NoArgs,
		RunE:  runFactory,
	}

	listBlueprintsCmd := &cobra.Command{
		Use:   "list-blueprints [--dir <path>]",
		Short: "List available blueprints",
		Long:  "List all available blueprint YAML files in the specified directory.",
		RunE:  listBlueprints,
	}

	runCmd.Flags().StringVarP(&blueprintPath, "blueprint", "b", "", "Path to blueprint YAML")
	runCmd.Flags().StringVarP(&task, "task", "t", "", "Task to execute")
	runCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Disable TUI, print progress to stdout")
	runCmd.MarkFlagRequired("blueprint")
	runCmd.MarkFlagRequired("task")

	listBlueprintsCmd.Flags().StringVarP(&blueprintsDir, "dir", "d", "./blueprints", "Directory containing blueprints")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listBlueprintsCmd)
}

func initV1Commands() {
	// Factory management
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new factory",
		RunE:  initializeFactory,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show factory status",
		RunE:  showFactoryStatus,
	}

	bootCmd := &cobra.Command{
		Use:   "boot",
		Short: "Start all stations",
		RunE:  bootFactory,
	}

	shutdownCmd := &cobra.Command{
		Use:   "shutdown",
		Short: "Graceful shutdown",
		RunE:  shutdownFactory,
	}

	pauseCmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause factory",
		RunE:  pauseFactory,
	}

	resumeCmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume factory",
		RunE:  resumeFactory,
	}

	// Stations
	stationCmd := &cobra.Command{
		Use:   "station",
		Short: "Station management commands",
	}

	addStationCmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Provision a new station",
		Args:  cobra.ExactArgs(1),
		RunE:  addStation,
	}

	listStationsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stations",
		RunE:  listStations,
	}

	removeStationCmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Decommission a station",
		Args:  cobra.ExactArgs(1),
		RunE:  removeStation,
	}

	stationStatusCmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show station status",
		Args:  cobra.ExactArgs(1),
		RunE:  showStationStatus,
	}

	stationCmd.AddCommand(addStationCmd, listStationsCmd, removeStationCmd, stationStatusCmd)

	// Operators
	operatorCmd := &cobra.Command{
		Use:   "operator",
		Short: "Operator management commands",
	}

	spawnOperatorCmd := &cobra.Command{
		Use:   "spawn <station>",
		Short: "Spawn an operator at a station",
		Args:  cobra.ExactArgs(1),
		RunE:  spawnOperator,
	}

	listOperatorsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all operators",
		RunE:  listOperators,
	}

	operatorStatusCmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show operator status",
		Args:  cobra.ExactArgs(1),
		RunE:  showOperatorStatus,
	}

	decommissionOperatorCmd := &cobra.Command{
		Use:   "decommission <id>",
		Short: "Decommission an operator",
		Args:  cobra.ExactArgs(1),
		RunE:  decommissionOperator,
	}

	operatorCmd.AddCommand(spawnOperatorCmd, listOperatorsCmd, operatorStatusCmd, decommissionOperatorCmd)

	// Work Cells
	cellCmd := &cobra.Command{
		Use:   "cell",
		Short: "Work cell management commands",
	}

	createCellCmd := &cobra.Command{
		Use:   "create <name> <stations...>",
		Short: "Create a work cell",
		Args:  cobra.MinimumNArgs(2),
		RunE:  createWorkCell,
	}

	activateCellCmd := &cobra.Command{
		Use:   "activate <cell-id>",
		Short: "Activate parallel execution",
		Args:  cobra.ExactArgs(1),
		RunE:  activateWorkCell,
	}

	cellStatusCmd := &cobra.Command{
		Use:   "status <cell-id>",
		Short: "Show cell status",
		Args:  cobra.ExactArgs(1),
		RunE:  showCellStatus,
	}

	disperseCellCmd := &cobra.Command{
		Use:   "disperse <cell-id>",
		Short: "Disperse cell",
		Args:  cobra.ExactArgs(1),
		RunE:  disperseWorkCell,
	}

	cellCmd.AddCommand(createCellCmd, activateCellCmd, cellStatusCmd, disperseCellCmd)

	// Work management (via beads CLI)
	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Job management commands",
	}

	createJobCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a job ticket (bead)",
		Args:  cobra.ExactArgs(1),
		RunE:  createJob,
	}

	listJobsCmd := &cobra.Command{
		Use:   "list",
		Short: "List job tickets",
		RunE:  listJobs,
	}

	showJobCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show ticket details",
		Args:  cobra.ExactArgs(1),
		RunE:  showJob,
	}

	closeJobCmd := &cobra.Command{
		Use:   "close <id>",
		Short: "Close a ticket",
		Args:  cobra.ExactArgs(1),
		RunE:  closeJob,
	}

	epicJobCmd := &cobra.Command{
		Use:   "epic <id>",
		Short: "Convert to epic",
		Args:  cobra.ExactArgs(1),
		RunE:  convertToEpic,
	}

	addChildCmd := &cobra.Command{
		Use:   "add-child <parent> <child>",
		Short: "Add child to epic",
		Args:  cobra.ExactArgs(2),
		RunE:  addChildToEpic,
	}

	jobCmd.AddCommand(createJobCmd, listJobsCmd, showJobCmd, closeJobCmd, epicJobCmd, addChildCmd)

	// Traveler management
	travelerCmd := &cobra.Command{
		Use:   "traveler",
		Short: "Traveler management commands",
	}

	attachTravelerCmd := &cobra.Command{
		Use:   "attach <station> <job>",
		Short: "Attach work to station",
		Args:  cobra.ExactArgs(2),
		RunE:  attachTraveler,
	}

	showTravelerCmd := &cobra.Command{
		Use:   "show <station>",
		Short: "Show station's traveler",
		Args:  cobra.ExactArgs(1),
		RunE:  showTraveler,
	}

	clearTravelerCmd := &cobra.Command{
		Use:   "clear <station>",
		Short: "Clear station's traveler",
		Args:  cobra.ExactArgs(1),
		RunE:  clearTraveler,
	}

	travelerCmd.AddCommand(attachTravelerCmd, showTravelerCmd, clearTravelerCmd)

	// Batch management
	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch management commands",
	}

	createBatchCmd := &cobra.Command{
		Use:   "create <name> <jobs...>",
		Short: "Create batch",
		Args:  cobra.MinimumNArgs(2),
		RunE:  createBatch,
	}

	batchStatusCmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show batch status",
		Args:  cobra.ExactArgs(1),
		RunE:  showBatchStatus,
	}

	listBatchesCmd := &cobra.Command{
		Use:   "list",
		Short: "List batches",
		RunE:  listBatches,
	}

	batchDashboardCmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Show batch dashboard (TUI)",
		RunE:  showBatchDashboard,
	}

	batchCmd.AddCommand(createBatchCmd, batchStatusCmd, listBatchesCmd, batchDashboardCmd)

	// SOPs & Formulas
	formulaCmd := &cobra.Command{
		Use:   "formula",
		Short: "Formula management commands",
	}

	loadFormulaCmd := &cobra.Command{
		Use:   "load <path>",
		Short: "Load a formula",
		Args:  cobra.ExactArgs(1),
		RunE:  loadFormula,
	}

	listFormulasCmd := &cobra.Command{
		Use:   "list",
		Short: "List available formulas",
		RunE:  listFormulas,
	}

	formulaCmd.AddCommand(loadFormulaCmd, listFormulasCmd)

	sopCmd := &cobra.Command{
		Use:   "sop",
		Short: "SOP management commands",
	}

	createSOPCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a SOP",
		Args:  cobra.ExactArgs(1),
		RunE:  createSOP,
	}

	sopStatusCmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show SOP status",
		Args:  cobra.ExactArgs(1),
		RunE:  showSOPStatus,
	}

	sopCmd.AddCommand(createSOPCmd, sopStatusCmd)

	// Execution
	runFormulaCmd := &cobra.Command{
		Use:   "run --formula <path> --task <task>",
		Short: "Run a formula",
		Args:  cobra.NoArgs,
		RunE:  runFormula,
	}

	runFormulaCmd.Flags().String("formula", "", "Path to formula file")
	runFormulaCmd.Flags().String("task", "", "Task to execute")
	runFormulaCmd.MarkFlagRequired("formula")
	runFormulaCmd.MarkFlagRequired("task")

	dispatchCmd := &cobra.Command{
		Use:   "dispatch <job> <station>",
		Short: "Dispatch work to station",
		Args:  cobra.ExactArgs(2),
		RunE:  dispatchWork,
	}

	// Support Service
	nudgeCmd := &cobra.Command{
		Use:   "nudge <operator>",
		Short: "Nudge operator to check traveler",
		Args:  cobra.ExactArgs(1),
		RunE:  nudgeOperator,
	}

	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Run health check",
		RunE:  runHealthCheck,
	}

	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Run cleanup",
		RunE:  runCleanup,
	}

	// Merge Queue
	mqCmd := &cobra.Command{
		Use:   "mq",
		Short: "Merge queue commands",
	}

	mqListCmd := &cobra.Command{
		Use:   "list",
		Short: "List merge queue",
		RunE:  listMergeQueue,
	}

	mqStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show merge queue status",
		RunE:  showMergeQueueStatus,
	}

	mqEscalateCmd := &cobra.Command{
		Use:   "escalate <mr-id>",
		Short: "Escalate MR",
		Args:  cobra.ExactArgs(1),
		RunE:  escalateMergeRequest,
	}

	mqCmd.AddCommand(mqListCmd, mqStatusCmd, mqEscalateCmd)

	// Mail
	mailCmd := &cobra.Command{
		Use:   "mail",
		Short: "Mail commands",
	}

	sendMailCmd := &cobra.Command{
		Use:   "send <to> <subject> <body>",
		Short: "Send mail to station",
		Args:  cobra.ExactArgs(3),
		RunE:  sendMail,
	}

	readMailCmd := &cobra.Command{
		Use:   "read",
		Short: "Read your mail",
		RunE:  readMail,
	}

	broadcastMailCmd := &cobra.Command{
		Use:   "broadcast <subject> <body>",
		Short: "Broadcast to all",
		Args:  cobra.ExactArgs(2),
		RunE:  broadcastMail,
	}

	mailCmd.AddCommand(sendMailCmd, readMailCmd, broadcastMailCmd)

	// Roles
	roleCmd := &cobra.Command{
		Use:   "role",
		Short: "Role management commands",
	}

	startRoleCmd := &cobra.Command{
		Use:   "start <role>",
		Short: "Start a role agent",
		Args:  cobra.ExactArgs(1),
		RunE:  startRole,
	}

	stopRoleCmd := &cobra.Command{
		Use:   "stop <role>",
		Short: "Stop a role agent",
		Args:  cobra.ExactArgs(1),
		RunE:  stopRole,
	}

	listRolesCmd := &cobra.Command{
		Use:   "list",
		Short: "List all roles and their status",
		RunE:  listRoles,
	}

	roleCmd.AddCommand(startRoleCmd, stopRoleCmd, listRolesCmd)

	// Add all commands to root
	rootCmd.AddCommand(initCmd, statusCmd, bootCmd, shutdownCmd, pauseCmd, resumeCmd)
	rootCmd.AddCommand(stationCmd, operatorCmd, cellCmd, jobCmd, travelerCmd, batchCmd)
	rootCmd.AddCommand(formulaCmd, sopCmd)
	rootCmd.AddCommand(runFormulaCmd, dispatchCmd)
	rootCmd.AddCommand(nudgeCmd, healthCmd, cleanupCmd, mqCmd, mailCmd, roleCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./configs/factory.yaml", "Path to factory config")
	rootCmd.PersistentFlags().StringVar(&projectPath, "project-path", ".", "Project path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// v0.x implementations
func runFactory(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	claudeBin := os.Getenv("CLAUDE_BIN")

	blueprint, err := config.LoadBlueprint(blueprintPath)
	if err != nil {
		return fmt.Errorf("failed to load blueprint: %w", err)
	}

	agent, err := agents.NewAgent("claude", claudeBin)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	f, err := factory.New(blueprint, agent)
	if err != nil {
		return fmt.Errorf("failed to create factory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	eventsChan := make(chan events.Event, 100)

	if noTUI {
		return runWithoutTUI(ctx, f, blueprint, task, eventsChan)
	}

	return runWithTUI(ctx, f, blueprint, task, eventsChan)
}

func runWithoutTUI(ctx context.Context, f *factory.Factory, blueprint *config.Blueprint, task string, eventsChan chan events.Event) error {
	fmt.Printf("🏭 Starting factory: %s\n", blueprint.Factory.Name)
	fmt.Println(strings.Repeat("─", 50))

	resultChan := make(chan *job.JobResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := f.Run(ctx, task, eventsChan)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	for {
		select {
		case evt := <-eventsChan:
			printEvent(evt)
			if evt.Type == events.EvtDone {
				fmt.Println("\n" + strings.Repeat("=", 50))
				fmt.Println("Final Output:")
				fmt.Println(strings.Repeat("=", 50))
				if output, ok := evt.Payload["output"].(string); ok {
					fmt.Println(output)
				}
				return nil
			}
		case result := <-resultChan:
			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("Final Output:")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Println(result.FinalOutput)
			return nil
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func printEvent(evt events.Event) {
	lineName := evt.Source
	stationName := evt.Subject
	if ln, ok := evt.Payload["line_name"].(string); ok {
		lineName = ln
	}
	if sn, ok := evt.Payload["station_name"].(string); ok {
		stationName = sn
	}

	switch evt.Type {
	case events.EvtStationStarted:
		fmt.Printf("[START]    %s / %s\n", lineName, stationName)
	case events.EvtStationInspecting:
		fmt.Printf("[INSPECT]  %s / %s\n", lineName, stationName)
	case events.EvtStationDone:
		retryStr := ""
		if retries, ok := evt.Payload["retries"].(int); ok && retries > 0 {
			retryStr = fmt.Sprintf(" (x%d)", retries+1)
		}
		duration := 0.0
		if d, ok := evt.Payload["duration"].(int64); ok {
			duration = float64(d) / 1000.0
		}
		fmt.Printf("[DONE]     %s / %s%s (%.1fs)\n",
			lineName, stationName, retryStr, duration)
	case events.EvtStationFailed:
		errMsg := "unknown error"
		if err, ok := evt.Payload["error"].(string); ok {
			errMsg = err
		}
		fmt.Printf("[FAIL]     %s / %s: %s\n", lineName, stationName, errMsg)
	case events.EvtMerging:
		fmt.Println("[MERGING]  Combining outputs...")
	}
}

func runWithTUI(ctx context.Context, f *factory.Factory, blueprint *config.Blueprint, task string, eventsChan chan events.Event) error {
	go func() {
		_, _ = f.Run(ctx, task, eventsChan)
	}()

	model := tui.NewModel(blueprint, eventsChan)
	p := tea.NewProgram(model)

	_, err := p.Run()
	return err
}

func listBlueprints(cmd *cobra.Command, args []string) error {
	blueprintsDir := "./blueprints"
	if dir := cmd.Flag("dir"); dir != nil && dir.Value.String() != "" {
		blueprintsDir = dir.Value.String()
	}

	entries, err := os.ReadDir(blueprintsDir)
	if err != nil {
		return fmt.Errorf("failed to read blueprints directory: %w", err)
	}

	fmt.Println("Available Blueprints:")
	fmt.Println(strings.Repeat("=", 50))

	found := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
			path := filepath.Join(blueprintsDir, entry.Name())
			bp, err := config.LoadBlueprint(path)
			if err != nil {
				fmt.Printf("\n%s: (error loading: %v)\n", entry.Name(), err)
				continue
			}

			found = true
			fmt.Printf("\n📄 %s\n", entry.Name())
			fmt.Printf("   Name: %s\n", bp.Factory.Name)
			fmt.Printf("   Description: %s\n", bp.Factory.Description)
			fmt.Printf("   Assembly Lines: %d\n", len(bp.Factory.AssemblyLines))
			for _, line := range bp.Factory.AssemblyLines {
				fmt.Printf("     - %s (%d stations)\n", line.Name, len(line.Stations))
			}
		}
	}

	if !found {
		fmt.Println("\nNo blueprint files found in", blueprintsDir)
	}

	return nil
}

// v1.0 command implementations (stubs for now)

func initializeFactory(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing factory...")
	// TODO: Create .factory directory, initialize database, etc.
	return nil
}

func showFactoryStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("Factory Status:")
	// TODO: Implement status display
	return nil
}

func bootFactory(cmd *cobra.Command, args []string) error {
	fmt.Println("Booting factory...")
	// TODO: Start all services
	return nil
}

func shutdownFactory(cmd *cobra.Command, args []string) error {
	fmt.Println("Shutting down factory...")
	// TODO: Graceful shutdown
	return nil
}

func pauseFactory(cmd *cobra.Command, args []string) error {
	fmt.Println("Pausing factory...")
	// TODO: Pause operations
	return nil
}

func resumeFactory(cmd *cobra.Command, args []string) error {
	fmt.Println("Resuming factory...")
	// TODO: Resume operations
	return nil
}

func addStation(cmd *cobra.Command, args []string) error {
	name := args[0]
	fmt.Printf("Adding station: %s\n", name)
	// TODO: Implement station provisioning
	return nil
}

func listStations(cmd *cobra.Command, args []string) error {
	fmt.Println("Stations:")
	// TODO: List stations
	return nil
}

func removeStation(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Removing station: %s\n", id)
	// TODO: Remove station
	return nil
}

func showStationStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Station status for: %s\n", id)
	// TODO: Show station status
	return nil
}

func spawnOperator(cmd *cobra.Command, args []string) error {
	station := args[0]
	fmt.Printf("Spawning operator at: %s\n", station)
	// TODO: Spawn operator
	return nil
}

func listOperators(cmd *cobra.Command, args []string) error {
	fmt.Println("Operators:")
	// TODO: List operators
	return nil
}

func showOperatorStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Operator status for: %s\n", id)
	// TODO: Show operator status
	return nil
}

func decommissionOperator(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Decommissioning operator: %s\n", id)
	// TODO: Decommission operator
	return nil
}

func createWorkCell(cmd *cobra.Command, args []string) error {
	name := args[0]
	stations := args[1:]
	fmt.Printf("Creating work cell '%s' with stations: %v\n", name, stations)
	// TODO: Create work cell
	return nil
}

func activateWorkCell(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Activating work cell: %s\n", id)
	// TODO: Activate work cell
	return nil
}

func showCellStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Work cell status for: %s\n", id)
	// TODO: Show cell status
	return nil
}

func disperseWorkCell(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Dispersing work cell: %s\n", id)
	// TODO: Disperse work cell
	return nil
}

func createJob(cmd *cobra.Command, args []string) error {
	title := strings.Join(args, " ")
	fmt.Printf("Creating job: %s\n", title)
	// TODO: Create job via beads client
	return nil
}

func listJobs(cmd *cobra.Command, args []string) error {
	fmt.Println("Jobs:")
	// TODO: List jobs via beads client
	return nil
}

func showJob(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Job details for: %s\n", id)
	// TODO: Show job details
	return nil
}

func closeJob(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Closing job: %s\n", id)
	// TODO: Close job
	return nil
}

func convertToEpic(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Converting to epic: %s\n", id)
	// TODO: Convert to epic
	return nil
}

func addChildToEpic(cmd *cobra.Command, args []string) error {
	parent := args[0]
	child := args[1]
	fmt.Printf("Adding %s as child of %s\n", child, parent)
	// TODO: Add child to epic
	return nil
}

func attachTraveler(cmd *cobra.Command, args []string) error {
	station := args[0]
	job := args[1]
	fmt.Printf("Attaching traveler: station=%s job=%s\n", station, job)
	// TODO: Attach traveler
	return nil
}

func showTraveler(cmd *cobra.Command, args []string) error {
	station := args[0]
	fmt.Printf("Traveler for station: %s\n", station)
	// TODO: Show traveler
	return nil
}

func clearTraveler(cmd *cobra.Command, args []string) error {
	station := args[0]
	fmt.Printf("Clearing traveler for station: %s\n", station)
	// TODO: Clear traveler
	return nil
}

func createBatch(cmd *cobra.Command, args []string) error {
	name := args[0]
	jobs := args[1:]
	fmt.Printf("Creating batch '%s' with jobs: %v\n", name, jobs)
	// TODO: Create batch
	return nil
}

func showBatchStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Batch status for: %s\n", id)
	// TODO: Show batch status
	return nil
}

func listBatches(cmd *cobra.Command, args []string) error {
	fmt.Println("Batches:")
	// TODO: List batches
	return nil
}

func showBatchDashboard(cmd *cobra.Command, args []string) error {
	fmt.Println("Batch Dashboard:")
	// TODO: Show TUI dashboard
	return nil
}

func loadFormula(cmd *cobra.Command, args []string) error {
	path := args[0]
	fmt.Printf("Loading formula: %s\n", path)
	// TODO: Load and validate formula
	return nil
}

func listFormulas(cmd *cobra.Command, args []string) error {
	fmt.Println("Available Formulas:")
	// TODO: List formulas from formulas/ directory
	return nil
}

func createSOP(cmd *cobra.Command, args []string) error {
	name := strings.Join(args, " ")
	fmt.Printf("Creating SOP: %s\n", name)
	// TODO: Create SOP
	return nil
}

func showSOPStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("SOP status for: %s\n", id)
	// TODO: Show SOP status
	return nil
}

func runFormula(cmd *cobra.Command, args []string) error {
	path, _ := cmd.Flags().GetString("formula")
	task, _ := cmd.Flags().GetString("task")
	fmt.Printf("Running formula: %s with task: %s\n", path, task)
	// TODO: Run formula
	return nil
}

func dispatchWork(cmd *cobra.Command, args []string) error {
	job := args[0]
	station := args[1]
	fmt.Printf("Dispatching job %s to station %s\n", job, station)
	// TODO: Dispatch work
	return nil
}

func nudgeOperator(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Nudging operator: %s\n", id)
	// TODO: Nudge operator
	return nil
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("Running health check...")
	// TODO: Run health check
	return nil
}

func runCleanup(cmd *cobra.Command, args []string) error {
	fmt.Println("Running cleanup...")
	// TODO: Run cleanup
	return nil
}

func listMergeQueue(cmd *cobra.Command, args []string) error {
	fmt.Println("Merge Queue:")
	// TODO: List merge queue
	return nil
}

func showMergeQueueStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("Merge Queue Status:")
	// TODO: Show merge queue status
	return nil
}

func escalateMergeRequest(cmd *cobra.Command, args []string) error {
	id := args[0]
	fmt.Printf("Escalating merge request: %s\n", id)
	// TODO: Escalate MR
	return nil
}

func sendMail(cmd *cobra.Command, args []string) error {
	to := args[0]
	subject := args[1]
	body := args[2]
	fmt.Printf("Sending mail to %s: %s\n", to, subject)
	_ = body
	// TODO: Send mail
	return nil
}

func readMail(cmd *cobra.Command, args []string) error {
	fmt.Println("Reading mail...")
	// TODO: Read mail
	return nil
}

func broadcastMail(cmd *cobra.Command, args []string) error {
	subject := args[0]
	body := args[1]
	fmt.Printf("Broadcasting: %s\n", subject)
	_ = body
	// TODO: Broadcast mail
	return nil
}

func startRole(cmd *cobra.Command, args []string) error {
	roleName := args[0]
	fmt.Printf("Starting role: %s\n", roleName)
	// TODO: Start role agent
	return nil
}

func stopRole(cmd *cobra.Command, args []string) error {
	roleName := args[0]
	fmt.Printf("Stopping role: %s\n", roleName)
	// TODO: Stop role agent
	return nil
}

func listRoles(cmd *cobra.Command, args []string) error {
	fmt.Println("Roles:")
	fmt.Println("  - director")
	fmt.Println("  - operator")
	fmt.Println("  - inspector")
	fmt.Println("  - supervisor")
	return nil
}
