package hotel_test

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	db "sigmaos/debug"
	"sigmaos/hotel"
	"sigmaos/loadgen"
	np "sigmaos/ninep"
	"sigmaos/perf"
	"sigmaos/proc"
	"sigmaos/protdevclnt"
	"sigmaos/protdevsrv"
	rd "sigmaos/rand"
	"sigmaos/test"
)

var K8S_ADDR string
var MAX_RPS int
var DURATION time.Duration

func init() {
	flag.StringVar(&K8S_ADDR, "k8saddr", "", "Addr of k8s frontend.")
	flag.IntVar(&MAX_RPS, "maxrps", 1000, "Max number of requests/sec.")
	flag.DurationVar(&DURATION, "duration", 10*time.Second, "Duration of load generation benchmarks.")
}

type Tstate struct {
	*test.Tstate
	job  string
	pids []proc.Tpid
}

func spawn(t *testing.T, ts *Tstate, srv, job string) proc.Tpid {
	p := proc.MakeProc(srv, []string{job})
	p.SetNcore(1)
	err := ts.Spawn(p)
	assert.Nil(t, err, "Spawn")
	err = ts.WaitStart(p.Pid)
	assert.Nil(t, err, "WaitStarted")
	return p.Pid
}

func makeTstate(t *testing.T, srvs []string) *Tstate {
	var err error
	ts := &Tstate{}
	ts.job = rd.String(8)
	ts.Tstate = test.MakeTstateAll(t)
	hotel.InitHotelFs(ts.FsLib, ts.job)
	ts.pids = make([]proc.Tpid, 0)
	for _, s := range srvs {
		pid := spawn(t, ts, s, ts.job)
		err = ts.WaitStart(pid)
		assert.Nil(t, err)
		ts.pids = append(ts.pids, pid)
	}
	return ts
}

func (ts *Tstate) Stats(fn string) {
	stats := &protdevsrv.Stats{}
	err := ts.GetFileJson(fn+"/"+protdevsrv.STATS, stats)
	assert.Nil(ts.T, err)
	fmt.Printf("= %s: %v\n", fn, stats)
}

func (ts *Tstate) stop() {
	for _, pid := range ts.pids {
		err := ts.Evict(pid)
		assert.Nil(ts.T, err, "Evict: %v", err)
		_, err = ts.WaitExit(pid)
		assert.Nil(ts.T, err)
	}
	sts, err := ts.GetDir(np.DBD)
	assert.Nil(ts.T, err)
	assert.Equal(ts.T, 5, len(sts))
}

func TestGeo(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-geod"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELGEO)
	assert.Nil(t, err)
	arg := hotel.GeoRequest{
		Lat: 37.7749,
		Lon: -122.4194,
	}
	res := &hotel.GeoResult{}
	err = pdc.RPC("Geo.Nearby", arg, &res)
	assert.Nil(t, err)
	db.DPrintf(db.ALWAYS, "res %v\n", res)
	assert.Equal(t, 5, len(res.HotelIds))
	ts.stop()
	ts.Shutdown()
}

func TestCache(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-cached"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELCACHE)
	assert.Nil(t, err)
	v := []byte("hello")
	arg := hotel.CacheRequest{
		Key:   "x",
		Value: v,
	}
	res := &hotel.CacheResult{}
	err = pdc.RPC("Cache.Set", arg, &res)
	assert.Nil(t, err)

	err = pdc.RPC("Cache.Get", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, v, res.Value)

	arg.Key = "y"
	err = pdc.RPC("Cache.Get", arg, &res)
	assert.NotNil(t, err)
	assert.Equal(t, hotel.ErrMiss, err)

	ts.stop()
	ts.Shutdown()
}

func TestCacheConcur(t *testing.T) {
	const N = 3
	ts := makeTstate(t, []string{"user/hotel-cached"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELCACHE)
	assert.Nil(t, err)
	v := []byte("hello")
	arg := hotel.CacheRequest{
		Key:   "x",
		Value: v,
	}
	res := &hotel.CacheResult{}
	err = pdc.RPC("Cache.Set", arg, &res)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			arg := hotel.CacheRequest{
				Key:   "x",
				Value: v,
			}
			res := &hotel.CacheResult{}
			err = pdc.RPC("Cache.Get", arg, res)
			assert.Nil(t, err)
			assert.Equal(t, v, res.Value)
			db.DPrintf("TEST", "Done get")
		}()
	}
	wg.Wait()

	ts.stop()
	ts.Shutdown()
}

func TestRate(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-cached", "user/hotel-rated"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELRATE)
	assert.Nil(t, err)
	arg := hotel.RateRequest{
		HotelIds: []string{"5", "3", "1", "6", "2"}, // from TestGeo
		InDate:   "2015-04-09",
		OutDate:  "2015-04-10",
	}
	var res hotel.RateResult
	err = pdc.RPC("Rate.GetRates", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res.RatePlans))
	err = pdc.RPC("Rate.GetRates", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res.RatePlans))
	ts.stop()
	ts.Shutdown()
}

func TestRec(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-recd"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELREC)
	assert.Nil(t, err)
	arg := hotel.RecRequest{
		Require: "dis",
		Lat:     38.0235,
		Lon:     -122.095,
	}
	var res hotel.RecResult
	err = pdc.RPC("Rec.GetRecs", arg, &res)
	assert.Nil(t, err)
	db.DPrintf(db.ALWAYS, "res %v\n", res.HotelIds)
	assert.Equal(t, 1, len(res.HotelIds))
	ts.stop()
	ts.Shutdown()
}

func TestUser(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-userd"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELUSER)
	assert.Nil(t, err)
	arg := hotel.UserRequest{
		Name:     "Cornell_0",
		Password: hotel.MkPassword("0"),
	}
	var res hotel.UserResult
	err = pdc.RPC("User.CheckUser", arg, &res)
	assert.Nil(t, err)
	db.DPrintf(db.ALWAYS, "res %v\n", res)
	ts.stop()
	ts.Shutdown()
}

func TestProfile(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-cached", "user/hotel-profd"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELPROF)
	assert.Nil(t, err)
	arg := hotel.ProfRequest{
		HotelIds: []string{"1", "2"},
	}
	var res hotel.ProfResult
	err = pdc.RPC("ProfSrv.GetProfiles", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res.Hotels))
	db.DPrintf(db.ALWAYS, "res %v\n", res.Hotels[0])

	err = pdc.RPC("ProfSrv.GetProfiles", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res.Hotels))

	ts.stop()
	ts.Shutdown()
}

func TestCheck(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-cached", "user/hotel-reserved"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELRESERVE)
	assert.Nil(t, err)
	arg := hotel.ReserveRequest{
		HotelId:      []string{"4"},
		CustomerName: "Cornell_0",
		InDate:       "2015-04-09",
		OutDate:      "2015-04-10",
		Number:       1,
	}
	var res hotel.ReserveResult
	err = pdc.RPC("Reserve.CheckAvailability", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res.HotelIds))
	err = pdc.RPC("Reserve.CheckAvailability", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res.HotelIds))
	ts.stop()
	ts.Shutdown()
}

func TestReserve(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-cached", "user/hotel-reserved"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELRESERVE)
	assert.Nil(t, err)
	arg := hotel.ReserveRequest{
		HotelId:      []string{"4"},
		CustomerName: "Cornell_0",
		InDate:       "2015-04-09",
		OutDate:      "2015-04-10",
		Number:       1,
	}
	var res hotel.ReserveResult

	err = pdc.RPC("Reserve.MakeReservation", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res.HotelIds))

	err = pdc.RPC("Reserve.MakeReservation", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res.HotelIds))

	ts.stop()
	ts.Shutdown()
}

func TestSingleSearch(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-geod", "user/hotel-cached", "user/hotel-rated", "user/hotel-searchd"})
	pdc, err := protdevclnt.MkProtDevClnt(ts.FsLib, np.HOTELSEARCH)
	assert.Nil(t, err)
	arg := hotel.SearchRequest{
		Lat:     37.7749,
		Lon:     -122.4194,
		InDate:  "2015-04-09",
		OutDate: "2015-04-10",
	}
	var res hotel.SearchResult
	err = pdc.RPC("Search.Nearby", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res.HotelIds))
	err = pdc.RPC("Search.Nearby", arg, &res)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res.HotelIds))
	ts.stop()
	ts.Shutdown()
}

func TestWww(t *testing.T) {
	ts := makeTstate(t, []string{"user/hotel-userd", "user/hotel-cached",
		"user/hotel-rated", "user/hotel-geod", "user/hotel-profd",
		"user/hotel-searchd", "user/hotel-reserved", "user/hotel-recd",
		"user/hotel-wwwd"})

	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)

	s, err := wc.Login("Cornell_0", hotel.MkPassword("0"))
	assert.Nil(t, err)
	assert.Equal(t, "Login successfully!", s)

	err = wc.Search("2015-04-09", "2015-04-10", 37.7749, -122.4194)
	assert.Nil(t, err)

	err = wc.Recs("dis", 38.0235, -122.095)
	assert.Nil(t, err)

	s, err = wc.Reserve("2015-04-09", "2015-04-10", 38.0235, -122.095, "1", "Cornell_0", "Cornell_0", hotel.MkPassword("0"), 1)
	assert.Nil(t, err)
	assert.Equal(t, "Reserve successfully!", s)

	ts.stop()
	ts.Shutdown()
}

func benchSearch(t *testing.T, wc *hotel.WebClnt, r *rand.Rand) {
	in_date := r.Intn(14) + 9
	out_date := in_date + r.Intn(5) + 1
	in_date_str := fmt.Sprintf("2015-04-%d", in_date)
	if in_date <= 9 {
		in_date_str = fmt.Sprintf("2015-04-0%d", in_date)
	}
	out_date_str := fmt.Sprintf("2015-04-%d", out_date)
	if out_date <= 9 {
		out_date_str = fmt.Sprintf("2015-04-0%d", out_date)
	}
	lat := 38.0235 + (float64(r.Intn(481))-240.5)/1000.0
	lon := -122.095 + (float64(r.Intn(325))-157.0)/1000.0
	err := wc.Search(in_date_str, out_date_str, lat, lon)
	assert.Nil(t, err, "Err search %v", err)
}

func benchRecommend(t *testing.T, wc *hotel.WebClnt, r *rand.Rand) {
	coin := toss(r)
	req := ""
	if coin < 0.33 {
		req = "dis"
	} else if coin < 0.66 {
		req = "rate"
	} else {
		req = "price"
	}
	lat := 38.0235 + (float64(r.Intn(481))-240.5)/1000.0
	lon := -122.095 + (float64(r.Intn(325))-157.0)/1000.0
	err := wc.Recs(req, lat, lon)
	assert.Nil(t, err)
}

func benchLogin(t *testing.T, wc *hotel.WebClnt, r *rand.Rand) {
	suffix := strconv.Itoa(r.Intn(500))
	user := "Cornell_" + suffix
	pw := hotel.MkPassword(suffix)
	s, err := wc.Login(user, pw)
	assert.Nil(t, err)
	assert.Equal(t, "Login successfully!", s)
}

func benchReserve(t *testing.T, wc *hotel.WebClnt, r *rand.Rand) {
	in_date := r.Intn(14) + 9
	out_date := in_date + r.Intn(5) + 1
	in_date_str := fmt.Sprintf("2015-04-%d", in_date)
	if in_date <= 9 {
		in_date_str = fmt.Sprintf("2015-04-0%d", in_date)
	}
	out_date_str := fmt.Sprintf("2015-04-%d", out_date)
	if out_date <= 9 {
		out_date_str = fmt.Sprintf("2015-04-0%d", out_date)
	}
	hotelid := strconv.Itoa(r.Intn(80) + 1)
	suffix := strconv.Itoa(r.Intn(500))
	user := "Cornell_" + suffix
	pw := hotel.MkPassword(suffix)
	cust_name := user
	num := 1
	lat := 38.0235 + (float64(r.Intn(481))-240.5)/1000.0
	lon := -122.095 + (float64(r.Intn(325))-157.0)/1000.0
	s, err := wc.Reserve(in_date_str, out_date_str, lat, lon, hotelid, user, cust_name, pw, num)
	assert.Nil(t, err)
	assert.Equal(t, "Reserve successfully!", s)
}

func toss(r *rand.Rand) float64 {
	toss := r.Intn(1000)
	return float64(toss) / 1000
}

var hotelsvcs = []string{"user/hotel-userd", "user/hotel-cached", "user/hotel-rated",
	"user/hotel-geod", "user/hotel-profd", "user/hotel-searchd",
	"user/hotel-reserved", "user/hotel-recd", "user/hotel-wwwd"}

//func TestStartAll(t *testing.T) {
//	ts := makeTstate(t, hotelsvcs)
//	addrs, err := hotel.GetJobHTTPAddrs(ts.FsLib, ts.job)
//	assert.Nil(t, err, "Err get http addr")
//	db.DPrintf(db.ALWAYS, "Setup done addrs %v", addrs)
//	for {
//	}
//}

func benchDSB(ts *Tstate, wc *hotel.WebClnt) {
	const (
		N               = 1000
		search_ratio    = 0.6
		recommend_ratio = 0.39
		user_ratio      = 0.005
		reserve_ratio   = 0.005
	)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()
	for i := 0; i < N; i++ {
		coin := toss(r)
		if coin < search_ratio {
			benchSearch(ts.T, wc, r)
		} else if coin < search_ratio+recommend_ratio {
			benchRecommend(ts.T, wc, r)
		} else if coin < search_ratio+recommend_ratio+user_ratio {
			benchLogin(ts.T, wc, r)
		} else {
			benchReserve(ts.T, wc, r)
		}
	}
	db.DPrintf(db.ALWAYS, "benchDSB N=%d %dms\n", N, time.Since(start).Milliseconds())
}

func TestBenchDeathStarSingle(t *testing.T) {
	ts := makeTstate(t, hotelsvcs)
	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)
	benchDSB(ts, wc)
	for _, s := range np.HOTELSVC {
		ts.Stats(s)
	}
	ts.stop()
	ts.Shutdown()
}

func TestBenchDeathStarSingleK8s(t *testing.T) {
	// Bail out if no addr was provided.
	if K8S_ADDR == "" {
		db.DPrintf(db.ALWAYS, "No k8s addr supplied")
		return
	}
	ts := makeTstate(t, nil)
	// Write a file for clients to discover the server's address.
	p := hotel.JobHTTPAddrsPath(ts.job)
	if err := ts.PutFileJson(p, 0777, []string{K8S_ADDR}); err != nil {
		db.DFatalf("Error PutFileJson addrs %v", err)
	}
	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)
	benchDSB(ts, wc)
	ts.Shutdown()
}

func TestBenchSearch(t *testing.T) {
	ts := makeTstate(t, hotelsvcs)
	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)
	p := perf.MakePerf("TEST")
	defer p.Done()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	lg := loadgen.MakeLoadGenerator(DURATION, MAX_RPS, func() {
		benchSearch(ts.T, wc, r)
	})
	lg.Run()
	for _, s := range np.HOTELSVC {
		ts.Stats(s)
	}
	ts.stop()
	ts.Shutdown()
}

func TestBenchSearchK8s(t *testing.T) {
	// Bail out if no addr was provided.
	if K8S_ADDR == "" {
		db.DPrintf(db.ALWAYS, "No k8s addr supplied")
		return
	}
	ts := makeTstate(t, nil)
	// Write a file for clients to discover the server's address.
	p := hotel.JobHTTPAddrsPath(ts.job)
	if err := ts.PutFileJson(p, 0777, []string{K8S_ADDR}); err != nil {
		db.DFatalf("Error PutFileJson addrs %v", err)
	}
	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)
	pf := perf.MakePerf("TEST")
	defer pf.Done()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	lg := loadgen.MakeLoadGenerator(DURATION, MAX_RPS, func() {
		benchSearch(ts.T, wc, r)
	})
	lg.Run()
	ts.Shutdown()
}

func testMultiSearch(t *testing.T, nthread int) {
	const (
		N = 1000
	)
	ts := makeTstate(t, hotelsvcs)
	wc := hotel.MakeWebClnt(ts.FsLib, ts.job)
	ch := make(chan bool)
	start := time.Now()
	for t := 0; t < nthread; t++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		go func() {
			for i := 0; i < N; i++ {
				benchSearch(ts.T, wc, r)
			}
			ch <- true
		}()
	}
	for t := 0; t < nthread; t++ {
		<-ch
	}
	db.DPrintf(db.ALWAYS, "TestBenchMultiSearch nthread=%d N=%d %dms\n", nthread, N, time.Since(start).Milliseconds())
	for _, s := range np.HOTELSVC {
		ts.Stats(s)
	}
	ts.stop()
	ts.Shutdown()
}

func TestMultiSearch(t *testing.T) {
	for _, n := range []int{1, 4} {
		testMultiSearch(t, n)
	}
}
