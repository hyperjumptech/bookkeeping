package middlewares

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/IDN-Media/awards/internal/config"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var (
	HMACAgeMinutes int
	SecretKey      string
)

func init() {
	HMACAgeMinutes = config.GetInt("hmac.age.minute")
	SecretKey = config.Get("hmac.secret")
}

func HMACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/docs") || (len(r.URL.Path) >= 10 && r.URL.Path[:10] == "/dashboard") || r.URL.Path == "/health" || r.URL.Path == "/devkey" {
			next.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		if len(header) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("you are not authorized"))
			return
		}
		hmacstr := strings.TrimSpace(header)
		if !ValidateHMAC(hmacstr) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("you are not authorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ComputeHmac(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func GenHMAC() string {
	time := time.Now().Format(time.RFC3339)
	hash := ComputeHmac(time, SecretKey)
	toBase := fmt.Sprintf("%s$%s", time, hash)
	base64hmac := base64.StdEncoding.EncodeToString([]byte(toBase))
	return base64hmac
}

func ValidateHMAC(hmac string) bool {
	decode, err := base64.StdEncoding.DecodeString(hmac)
	if err != nil {
		return false
	}
	splt := strings.Split(string(decode), "$")
	timeStr := splt[0]
	signature64 := splt[1]
	timeToCheck, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return false
	}
	if time.Now().Add((-1 * time.Duration(HMACAgeMinutes)) * time.Minute).After(timeToCheck) {
		return false
	}

	signature := ComputeHmac(timeStr, SecretKey)
	if signature64 != signature {
		return false
	}
	return true
}

// DevKey can be invoked from curl -X PUT -H "HocusPocus: AvadaCadavra" http://localhost:50051/devkey
func DevKey(w http.ResponseWriter, r *http.Request) {
	hocuspocus := r.Header.Get("HocusPocus")
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	gHMAC := GenHMAC()
	if hocuspocus == "AvadaCadavra" {
		w.Write([]byte(fmt.Sprintf("Looking for magical incantation.... Found\n")))
		first := rand.Intn(4)
		for i := 0; i < first; i++ {
			w.Write([]byte(fmt.Sprintf("%s\n", MakeResidue(gHMAC))))
		}
		w.Write([]byte(fmt.Sprintf("%s\n", gHMAC)))
		for i := 0; i < 4-first; i++ {
			w.Write([]byte(fmt.Sprintf("%s\n", MakeResidue(gHMAC))))
		}
	} else if len(hocuspocus) > 0 {
		w.Write([]byte(fmt.Sprintf("Looking for magical incantation.... Not Found : nothing happen")))
	} else {
		w.Write([]byte("not found"))
	}
}

const (
	RWords = "ButImustexplaintoyouhowallthismistakenideaofdenouncingpleasureandpraisingpainwasbornandIwillgiveyouacompleteaccountofthesystemandexpoundtheactualteachingsofthegreatexplorerofthetruththemasterbuilderofhumanhappinessNoonerejectsdislikesoravoidspleasureitselfbecauseitispleasurebutbecausethosewhodonotknowhowtopursuepleasurerationallyencounterconsequencesthatareextremelypainfulNoragainisthereanyonewholovesorpursuesordesirestoobtainpainofitselfbecauseitispainbutbecauseoccasionallycircumstancesoccurinwhichtoilandpaincanprocurehimsomegreatpleasureTotakeatrivialexamplewhichofuseverundertakeslaboriousphysicalexerciseexcepttoobtainsomeadvantagefromitButwhohasanyrighttofindfaultwithamanwhochoosestoenjoyapleasurethathasnoannoyingconsequencesoronewhoavoidsapainthatproducesnoresultantpleasure"
)

func MakeResidue(hmac string) string {
	l := len(hmac) - 2
	off := len(RWords) - (len(hmac) - 2)
	off = rand.Intn(off)
	torandUp := []byte(strings.ToUpper(RWords)[off : off+l])
	torandDw := []byte(strings.ToLower(RWords)[off : off+l])
	res := make([]byte, len(torandUp))
	for i, c := range torandUp {
		var nc byte
		switch rand.Intn(3) {
		case 0: // upper
			nc = c
		case 1: // lower
			nc = torandDw[i]
		default: // numbers
			switch c {
			case 'A':
				nc = '4'
			case 'I', 'L':
				nc = '1'
			case 'E':
				nc = '3'
			case 'S':
				nc = '5'
			case 'G':
				nc = '9'
			case 'O':
				nc = '0'
			default:
				nc = c
			}
		}
		res[i] = nc
	}
	return string(res) + "=="
}
