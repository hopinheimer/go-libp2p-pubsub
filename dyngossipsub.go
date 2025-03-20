type DynamicGossipsub struct {
    *pubsub.GossipSub
    paramsMutex sync.RWMutex
    currentParams pubsub.GossipSubParams
    floodPublish bool
}

func NewDynamicGossipSub(ctx context.Context, h host.Host, opts ...pubsub.Option) *DynamicGossipsub{
    gs := pubsub.NewDynamicGossipSub(ctx, h, opts...)
    return &DynamicGossipsub{
        GossipSub: gs,
        currentParams: pubsub.DefaultGossipSubParams(),
    }
}

func (dgs *DynamicGossipSub) SetParams(params pubsub.GossipSubParams) {
    dgs.paramsMutex.Lock()
    defer dgs.paramsMutex.Unlock()
    dgs.currentParams = params
    dgs.GossipSub.Heartbeat.Reset(time.Duration(params.Heartbeat) * time.Millisecond)
    
}

func (dgs *DynamicGossipSub) SetFloodPublish(flood bool) {
    dgs.paramsMutex.Lock()
    defer dgs.paramsMutex.Unlock()
    dgs.floodPublish = flood
}

func (dgs *DynamicGossipSub) GetParams() pubsub.GossipSubParams {
    dgs.paramsMutex.RLock()
    defer dgs.paramsMutex.RUnlock()
    return dgs.currentParams
}

func (dgs *DynamicGossipSub) enoughPeers(topic string, count int) bool{
    dgs.paramsMutex.RLock()
    defer dgs.paramsMutex.RUnlock()

    return count >= dgs.currentParams.Dlo && count <= dgs.currentParams.Dhi
}

func (dgs *DynamicGossipSub) heartbeat() {
    dgs.paramsMutex.RLock()
    interval := time.Duration(dgs.currentParams.Heartbeat) * time.Millisecond
    dgs.paramsMutex.RUnlock()
    
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            dgs.doHeartbeat()
        case <-dgs.ctx.Done():
            return
        }
    }
} 

func (dgs *DynamicGossipSub) handleUpdateParams(w http.ResponseWriter, r *http.Request) {
    var newParams pubsub.GossipSubParams
    if err := json.NewDecoder(r.Body).Decode(&newParams); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate parameters
    if newParams.Dlo < 0 || newParams.Dhi < newParams.Dlo {
        http.Error(w, "invalid parameters", http.StatusBadRequest)
        return
    }

    dgs.SetParams(newParams)
    w.WriteHeader(http.StatusOK)
}
