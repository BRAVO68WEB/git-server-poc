package observability

// import (
// 	"fmt"
// 	"net/http"
// 	"sync/atomic"
// 	"time"
// )

// var uploadPackRequests uint64
// var receivePackRequests uint64
// var uploadPackErrors uint64
// var receivePackErrors uint64
// var uploadPackDurationMs uint64
// var receivePackDurationMs uint64

// func RegisterMetrics(mux *http.ServeMux) {
// 	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
// 		fmt.Fprintf(w, "githut_upload_pack_requests %d\n", atomic.LoadUint64(&uploadPackRequests))
// 		fmt.Fprintf(w, "githut_receive_pack_requests %d\n", atomic.LoadUint64(&receivePackRequests))
// 		fmt.Fprintf(w, "githut_upload_pack_errors %d\n", atomic.LoadUint64(&uploadPackErrors))
// 		fmt.Fprintf(w, "githut_receive_pack_errors %d\n", atomic.LoadUint64(&receivePackErrors))
// 		fmt.Fprintf(w, "githut_upload_pack_duration_ms_total %d\n", atomic.LoadUint64(&uploadPackDurationMs))
// 		fmt.Fprintf(w, "githut_receive_pack_duration_ms_total %d\n", atomic.LoadUint64(&receivePackDurationMs))
// 	})
// }

// func MetricsHTTPHandler() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
// 		fmt.Fprintf(w, "githut_upload_pack_requests %d\n", atomic.LoadUint64(&uploadPackRequests))
// 		fmt.Fprintf(w, "githut_receive_pack_requests %d\n", atomic.LoadUint64(&receivePackRequests))
// 		fmt.Fprintf(w, "githut_upload_pack_errors %d\n", atomic.LoadUint64(&uploadPackErrors))
// 		fmt.Fprintf(w, "githut_receive_pack_errors %d\n", atomic.LoadUint64(&receivePackErrors))
// 		fmt.Fprintf(w, "githut_upload_pack_duration_ms_total %d\n", atomic.LoadUint64(&uploadPackDurationMs))
// 		fmt.Fprintf(w, "githut_receive_pack_duration_ms_total %d\n", atomic.LoadUint64(&receivePackDurationMs))
// 	}
// }

// func RecordUploadPack(d time.Duration, err bool) {
// 	atomic.AddUint64(&uploadPackRequests, 1)
// 	atomic.AddUint64(&uploadPackDurationMs, uint64(d.Milliseconds()))
// 	if err {
// 		atomic.AddUint64(&uploadPackErrors, 1)
// 	}
// }

// func RecordReceivePack(d time.Duration, err bool) {
// 	atomic.AddUint64(&receivePackRequests, 1)
// 	atomic.AddUint64(&receivePackDurationMs, uint64(d.Milliseconds()))
// 	if err {
// 		atomic.AddUint64(&receivePackErrors, 1)
// 	}
// }
