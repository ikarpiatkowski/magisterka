package main

import (
	"context"
	"sync"
	"time"
)

func runTest(cfg *Config, dbType string, m *metrics) {
       ctx, done := context.WithCancel(context.Background())
       defer done()

       var pg *postgres
       var mg *mongodb
       var es *elastic
       switch dbType {
       case "pg":
           pg = NewPostgres(cfg)
       case "mg":
           mg = NewMongo(cfg)
       case "es":
           es, _ = NewElasticsearch(ctx, cfg, m)
       }

       sleepInterval := time.Duration(cfg.Test.RequestDelayMs) * time.Millisecond
       for currentClients := cfg.Test.MinClients; currentClients <= cfg.Test.MaxClients; currentClients++ {
           m.clients.WithLabelValues(dbType, "stage").Set(float64(currentClients))
           stageCtx, cancelStage := context.WithCancel(ctx)
           var stageWG sync.WaitGroup
           for i := 0; i < currentClients; i++ {
               stageWG.Add(1)
               go func() {
                   defer stageWG.Done()
                   for {
                       select {
                       case <-stageCtx.Done():
                           return
                       default:
                       }
                       p1 := product{
                           Price:       float32(random(1, 100)),
                           TextContent: generateFTSContent(),
                       }
                       p2 := product{
                           Price:       float32(random(1, 100)),
                           TextContent: generateFTSContent(),
                       }

                       _ = p1.create(pg, mg, es, dbType, m)
                       _ = p2.create(pg, mg, es, dbType, m)

                       p1.Price = float32(random(1, 100))
                       _ = p1.update(pg, mg, es, dbType, m)

                       _ = p1.searchFTS(pg, mg, es, dbType, m)

                       _ = p2.delete(pg, mg, es, dbType, m)

                       if sleepInterval > 0 {
                           select {
                           case <-time.After(sleepInterval):
                           case <-stageCtx.Done():
                           }
                       }
                   }
               }()
           }
           time.Sleep(time.Duration(cfg.Test.StageIntervalS) * time.Second)
           cancelStage()
           stageWG.Wait()
       }
}