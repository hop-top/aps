package a2a

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"sync"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	eventqueue "github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/a2aproject/a2a-go/log"

	"oss-aps-cli/internal/core"
)

var _ a2asrv.RequestHandler = (*Server)(nil)

type Server struct {
	profileID    string
	profile      *core.Profile
	storage      *Storage
	executor     *Executor
	queueManager eventqueue.Manager
	httpServer   *http.Server
	mu           sync.RWMutex
	running      bool
	config       *StorageConfig
	pushConfigs  map[string]*a2a.TaskPushConfig
}

func NewServer(profile *core.Profile, config *StorageConfig) (*Server, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}

	if profile.A2A == nil || !profile.A2A.Enabled {
		return nil, ErrA2ANotEnabled
	}

	if config == nil {
		return nil, fmt.Errorf("storage config cannot be nil")
	}

	storage, err := NewStorage(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	executor := NewExecutor(profile, storage)
	queueManager := eventqueue.NewInMemoryManager()

	return &Server{
		profileID:    profile.ID,
		profile:      profile,
		storage:      storage,
		executor:     executor,
		queueManager: queueManager,
		config:       config,
		running:      false,
		pushConfigs:  make(map[string]*a2a.TaskPushConfig),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	handler := a2asrv.NewHandler(
		s.executor,
		a2asrv.WithTaskStore(s.storage),
		a2asrv.WithCallInterceptor(s),
	)

	mux := http.NewServeMux()
	mux.Handle("/", a2asrv.NewJSONRPCHandler(handler))
	mux.Handle("/.well-known/agent-card", s.getAgentCardHandler())

	addr := s.getAddress()
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.running = true

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "HTTP server error", err)
		}
	}()

	log.Info(ctx, "A2A server started", "profile_id", s.profileID, "address", addr)

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.httpServer != nil {
		ctx := context.Background()
		s.httpServer.Shutdown(ctx)
	}

	s.running = false

	log.Info(context.Background(), "A2A server stopped", "profile_id", s.profileID)

	return nil
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Server) ProfileID() string {
	return s.profileID
}

func (s *Server) GetStorage() *Storage {
	return s.storage
}

func (s *Server) Before(ctx context.Context, callCtx *a2asrv.CallContext, req *a2asrv.Request) (context.Context, error) {
	log.Info(ctx, "A2A request received", "method", callCtx.Method(), "profile_id", s.profileID)
	return ctx, nil
}

func (s *Server) After(ctx context.Context, callCtx *a2asrv.CallContext, resp *a2asrv.Response) error {
	return nil
}

func (s *Server) OnGetTask(ctx context.Context, query *a2a.TaskQueryParams) (*a2a.Task, error) {
	task, _, err := s.storage.Get(ctx, a2a.TaskID(query.ID))
	if err != nil {
		return nil, err
	}

	if query.HistoryLength != nil && *query.HistoryLength > 0 && len(task.History) > *query.HistoryLength {
		task.History = task.History[len(task.History)-*query.HistoryLength:]
	}

	return task, nil
}

func (s *Server) OnCancelTask(ctx context.Context, id *a2a.TaskIDParams) (*a2a.Task, error) {
	taskID := a2a.TaskID(id.ID)

	task, _, err := s.storage.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}

	queue, err := s.queueManager.GetOrCreate(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	defer queue.Close()

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		StoredTask: task,
		Message:    nil,
	}

	if err := s.executor.Cancel(ctx, reqCtx, queue); err != nil {
		return nil, fmt.Errorf("cancel failed: %w", err)
	}

	resultTask, _, err := s.storage.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return resultTask, nil
}

func (s *Server) OnSendMessage(ctx context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	taskID := a2a.NewTaskID()

	if params.Message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	if params.Message.TaskID != "" {
		taskID = params.Message.TaskID
	}

	task, _, err := s.storage.Get(ctx, taskID)
	storedTask := task
	if err != nil && err != a2a.ErrTaskNotFound {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	queue, err := s.queueManager.GetOrCreate(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	defer queue.Close()

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    params.Message,
		StoredTask: storedTask,
	}

	if storedTask != nil {
		reqCtx.RelatedTasks = []*a2a.Task{storedTask}
	}

	if err := s.executor.Execute(ctx, reqCtx, queue); err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	resultTask, _, err := s.storage.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve result task: %w", err)
	}

	return resultTask, nil
}

func (s *Server) OnSendMessageStream(ctx context.Context, params *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	taskID := a2a.NewTaskID()

	if params.Message == nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("message cannot be nil"))
		}
	}

	if params.Message.TaskID != "" {
		taskID = params.Message.TaskID
	}

	task, _, err := s.storage.Get(ctx, taskID)
	storedTask := task
	if err != nil && err != a2a.ErrTaskNotFound {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("failed to get task: %w", err))
		}
	}

	queue, err := s.queueManager.GetOrCreate(ctx, taskID)
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("failed to get queue: %w", err))
		}
	}

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    params.Message,
		StoredTask: storedTask,
	}

	if storedTask != nil {
		reqCtx.RelatedTasks = []*a2a.Task{storedTask}
	}

	return func(yield func(a2a.Event, error) bool) {
		defer queue.Close()

		execCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			if err := s.executor.Execute(execCtx, reqCtx, queue); err != nil {
				yield(nil, err)
				return
			}
		}()

		for {
			event, _, err := queue.Read(execCtx)
			if err != nil {
				if err == eventqueue.ErrQueueClosed {
					return
				}
				yield(nil, err)
				return
			}

			if !yield(event, nil) {
				return
			}
		}
	}
}

func (s *Server) OnResubscribeToTask(ctx context.Context, id *a2a.TaskIDParams) iter.Seq2[a2a.Event, error] {
	taskID := a2a.TaskID(id.ID)

	queue, err := s.queueManager.GetOrCreate(ctx, taskID)
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("failed to get queue: %w", err))
		}
	}

	return func(yield func(a2a.Event, error) bool) {
		defer queue.Close()

		for {
			event, _, err := queue.Read(ctx)
			if err != nil {
				if err == eventqueue.ErrQueueClosed {
					return
				}
				yield(nil, err)
				return
			}

			if !yield(event, nil) {
				return
			}
		}
	}
}

func (s *Server) OnGetTaskPushConfig(ctx context.Context, params *a2a.GetTaskPushConfigParams) (*a2a.TaskPushConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.pushConfigs[string(params.TaskID)]
	if !exists {
		return nil, fmt.Errorf("push config not found for task %s", params.TaskID)
	}

	return config, nil
}

func (s *Server) OnListTaskPushConfig(ctx context.Context, params *a2a.ListTaskPushConfigParams) ([]*a2a.TaskPushConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]*a2a.TaskPushConfig, 0, len(s.pushConfigs))
	for _, config := range s.pushConfigs {
		configs = append(configs, config)
	}

	return configs, nil
}

func (s *Server) OnSetTaskPushConfig(ctx context.Context, params *a2a.TaskPushConfig) (*a2a.TaskPushConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushConfigs[string(params.TaskID)] = params

	log.Info(ctx, "Task push config set", "task_id", params.TaskID, "url", params.Config.URL)

	return params, nil
}

func (s *Server) OnDeleteTaskPushConfig(ctx context.Context, params *a2a.DeleteTaskPushConfigParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pushConfigs, string(params.TaskID))

	log.Info(ctx, "Task push config deleted", "task_id", params.TaskID)

	return nil
}

func (s *Server) OnGetExtendedAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	card, err := GenerateAgentCardFromProfile(s.profile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent card: %w", err)
	}
	return card, nil
}

func (s *Server) getAddress() string {
	if s.profile.A2A.ListenAddr != "" {
		return s.profile.A2A.ListenAddr
	}
	return "127.0.0.1:8081"
}

func (s *Server) getAgentCardHandler() http.Handler {
	card, err := GenerateAgentCardFromProfile(s.profile)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, fmt.Sprintf("Failed to generate agent card: %v", err), http.StatusInternalServerError)
		})
	}

	return a2asrv.NewStaticAgentCardHandler(card)
}

