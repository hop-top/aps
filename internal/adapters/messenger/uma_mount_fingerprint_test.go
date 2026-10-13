package messenger

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func (r *umaRoutes) bridgeOf(id string) *umaInstance {
	r.adapter.uma.mu.Lock()
	defer r.adapter.uma.mu.Unlock()
	return r.adapter.uma.bridges[id]
}

// TestUmaRoute_DeliveryBookkeepingKeepsTheBridge: the runtime writes delivery
// status back to the service after every reply. That is bookkeeping, not
// configuration: the next webhook must reach the same engine, with its
// in-memory event de-duplication and replay window intact.
func TestUmaRoute_DeliveryBookkeepingKeepsTheBridge(t *testing.T) {
	r := newUmaRoutes(t)
	require.NoError(t, core.SaveService(umaSMSService("sms-uma")))

	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000021", umaSender, "one"), umaHookURL).Code)
	umaWait(t, "delivery recorded on the service", func() bool {
		svc, err := core.LoadService("sms-uma")
		return err == nil && svc.Delivery != nil && len(svc.Delivery.Attempts) > 0
	})
	first := r.bridgeOf("sms-uma")
	require.NotNil(t, first)

	require.Equal(t, http.StatusOK, r.postForm(t, "sms-uma", umaIncoming("SM10000000000000000000000000000022", umaSender, "two"), umaHookURL).Code)
	assert.Same(t, first, r.bridgeOf("sms-uma"), "delivery bookkeeping rebuilt the bridge")
}
