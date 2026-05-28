package main

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/app"
	"github.com/yunerou/niarb/app/server"
	"github.com/yunerou/niarb/app/task"
	exampleDeli "github.com/yunerou/niarb/internal/example/delivery"
	cacheprovider "github.com/yunerou/niarb/provider/cache-provider"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	eventslogprovider "github.com/yunerou/niarb/provider/events-log"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	healthyprovider "github.com/yunerou/niarb/provider/healthy-provider"
	llmchat "github.com/yunerou/niarb/provider/llm-chat"
	geminiaistudiochatadapter "github.com/yunerou/niarb/provider/llm-chat/gemini-aistudio-adapter"
	geminivertexchatadapter "github.com/yunerou/niarb/provider/llm-chat/gemini-vertex-adapter"
	openaichatadapter "github.com/yunerou/niarb/provider/llm-chat/openai-adapter"
	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
	geminiaistudioadapter "github.com/yunerou/niarb/provider/llm-embedding/gemini-aistudio-adapter"
	geminivertexadapter "github.com/yunerou/niarb/provider/llm-embedding/gemini-vertex-adapter"
	openaiadapter "github.com/yunerou/niarb/provider/llm-embedding/openai-adapter"
	muxmiddleware "github.com/yunerou/niarb/provider/mux-middleware"
	observerprovider "github.com/yunerou/niarb/provider/observer-provider"
	otelprovider "github.com/yunerou/niarb/provider/otel-provider"
	"github.com/yunerou/niarb/provider/postgres"
	pubsubvalkey "github.com/yunerou/niarb/provider/pubsub/valkey"
	qdrant "github.com/yunerou/niarb/provider/qdrant"
	queuevalkey "github.com/yunerou/niarb/provider/queue/valkey"
	s3provider "github.com/yunerou/niarb/provider/s3"
	validationprovider "github.com/yunerou/niarb/provider/validation-provider"
	workerpoolprovider "github.com/yunerou/niarb/provider/worker-pool-provider"
)

func registerProviders(i do.Injector) {
	// --- primitives & config ---
	do.ProvideValue(i, slog.Default())
	do.Provide(i, fncollector.NewDICleanupTask)
	do.Provide(i, fncollector.NewDIIntervalTask)
	do.Provide(i, validationprovider.NewDI)

	cfg := configprovider.FromYaml([]string{configPath})
	do.ProvideValue(i, cfg)

	// --- infrastructure (config-dependent) ---
	do.Provide(i, func(ix do.Injector) (*pgxpool.Pool, error) {
		c := do.MustInvoke[configprovider.ConfigStore](ix)
		return postgres.New(c), nil
	})
	do.Provide(i, healthyprovider.NewDI)
	do.Provide(i, muxmiddleware.NewDI)
	do.Provide(i, cacheprovider.NewDI)
	do.Provide(i, workerpoolprovider.NewDI)
	do.Provide(i, otelprovider.NewDI)
	do.Provide(i, queuevalkey.QueueValkeyDI)
	do.Provide(i, pubsubvalkey.PubsubValkeyDI)
	do.Provide(i, s3provider.NewDI)

	do.Provide(i, exampleDeli.NewDI)

	// --- vector + embedding providers ---
	do.Provide(i, func(ix do.Injector) (qdrant.QdrantProvider, error) {
		c := do.MustInvoke[configprovider.ConfigStore](ix)
		return qdrant.New(c.Env().Qdrant), nil
	})
	do.Provide(i, func(ix do.Injector) (llmembedding.EmbeddingProvider, error) {
		c := do.MustInvoke[configprovider.ConfigStore](ix)
		cfg := c.Env().LLMEmbedding
		var adapter llmembedding.Adapter
		switch cfg.Provider {
		case "", "openai":
			adapter = openaiadapter.New(cfg)
		case "gemini-aistudio":
			adapter = geminiaistudioadapter.New(cfg)
		case "gemini-vertex":
			adapter = geminivertexadapter.New(cfg)
		default:
			return nil, fmt.Errorf("llm-embedding: unknown provider %q", cfg.Provider)
		}
		return llmembedding.New(adapter), nil
	})
	do.Provide(i, func(ix do.Injector) (llmchat.Provider, error) {
		c := do.MustInvoke[configprovider.ConfigStore](ix)
		cfg := c.Env().LLMChat
		var adapter llmchat.Adapter
		switch cfg.Provider {
		case "", "openai":
			adapter = openaichatadapter.New(cfg)
		case "gemini-aistudio":
			adapter = geminiaistudiochatadapter.New(cfg)
		case "gemini-vertex":
			adapter = geminivertexchatadapter.New(cfg)
		default:
			return nil, fmt.Errorf("llm-chat: unknown provider %q", cfg.Provider)
		}
		return llmchat.New(adapter), nil
	})

	// --- observability & events ---
	do.Provide(i, observerprovider.NewDI)
	do.Provide(i, eventslogprovider.NewDI(eventslogprovider.Config{
		EnableSlog: true,
	}))

	// --- app commands ---
	do.Provide(i, func(ix do.Injector) (*server.SvCmd, error) {
		return server.NewSvCmd(ix), nil
	})
	do.Provide(i, func(ix do.Injector) (*task.TaskCmd, error) {
		return task.NewTaskCmd(ix), nil
	})
	do.Provide(i, func(ix do.Injector) (*app.CmdApp, error) {
		return app.NewCmdApp(ix), nil
	})
}
