package autoclaw

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/transport"
)

// Background manages background goroutines for the autoclaw provider:
// wallet balance checking and proactive token refresh.
type Background struct {
	doer   transport.Doer
	save   CredSaver
	getAll func() []provider.Account // returns current accounts from store

	mu            sync.Mutex
	exhausted     map[string]bool       // email → exhausted
	walletCache   map[string]float64    // email → balance
	lastWalletAll map[string]time.Time  // email → last wallet check
}

// NewBackground creates a background manager.
func NewBackground(doer transport.Doer, save CredSaver, getAll func() []provider.Account) *Background {
	return &Background{
		doer:          doer,
		save:          save,
		getAll:        getAll,
		exhausted:     map[string]bool{},
		walletCache:   map[string]float64{},
		lastWalletAll: map[string]time.Time{},
	}
}

// Start launches all background goroutines. Call once during server startup.
func (b *Background) Start() {
	go b.walletCheckerLoop()
	go b.tokenRefresherLoop()
	log.Println("[autoclaw] Background routines started")
}

// walletCheckerLoop checks all accounts' wallet balance every 5 minutes.
func (b *Background) walletCheckerLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()

	// Initial check after a short delay
	time.Sleep(10 * time.Second)
	b.checkAllWallets()

	for range t.C {
		b.checkAllWallets()
	}
}

func (b *Background) checkAllWallets() {
	accs := b.getAll()
	for _, acc := range accs {
		email := acc.Creds["email"]

		// Build an authManager on the fly just for wallet check
		am := newAuthManager(b.doer, b.save, acc)
		token, err := am.token()
		if err != nil {
			continue
		}

		balance, err := b.checkSingleWallet(token)
		if err != nil {
			continue
		}

		b.mu.Lock()
		b.walletCache[email] = balance
		b.lastWalletAll[email] = time.Now()
		b.exhausted[email] = balance <= 0
		b.mu.Unlock()
	}
}

// checkSingleWallet calls the wallet API for one account.
func (b *Background) checkSingleWallet(token string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, walletURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header = signedHeaders()
	req.Header.Set("authorization", "Bearer "+token)

	resp, err := b.doer.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var wr walletResp
	if err := json.Unmarshal(raw, &wr); err != nil {
		return 0, err
	}
	if wr.Code != 0 || wr.Data == nil {
		return 0, nil
	}
	return float64(wr.Data.TotalBalance), nil
}

// IsExhausted checks if an account was recently found exhausted (cached).
func (b *Background) IsExhausted(email string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exhausted[email]
}

// GetWalletBalance returns the cached wallet balance for an account.
func (b *Background) GetWalletBalance(email string) (float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	bal, ok := b.walletCache[email]
	return bal, ok
}

// GetAllWalletBalances returns all cached wallet balances for the dashboard.
func (b *Background) GetAllWalletBalances() map[string]float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[string]float64, len(b.walletCache))
	for k, v := range b.walletCache {
		out[k] = v
	}
	return out
}

// tokenRefresherLoop proactively refreshes all expiring tokens every 15 minutes.
func (b *Background) tokenRefresherLoop() {
	t := time.NewTicker(15 * time.Minute)
	defer t.Stop()

	time.Sleep(5 * time.Second)
	b.refreshAllTokens()

	for range t.C {
		b.refreshAllTokens()
	}
}

func (b *Background) refreshAllTokens() {
	accs := b.getAll()
	for _, acc := range accs {
		am := newAuthManager(b.doer, b.save, acc)
		_, err := am.token() // triggers refresh if needed
		if err != nil {
			log.Printf("[autoclaw] Background refresh failed for account %d: %v", acc.ID, err)
		}
	}
}
