//go:build darwin

package svc

import (
	"sync"
	"unicode/utf8"

	l "tmaxsrv/log"
)

const (
	retailServiceId = int64(999999999)

	respDetailList = ScaleMgrRespMsgType("resp_detail_list")
	respNewDetail  = ScaleMgrRespMsgType("resp_new_detail")
	respDetailAdd  = ScaleMgrRespMsgType("resp_detail_add")

	retailPakMaxBytes = 32 * 1024
)

var (
	retailProviderOnce sync.Once
	retailProvider     *DetailRecProvider

	retailMu           sync.Mutex
	retailLastPushedId uint
)

type retailPak struct {
	PakCount int
	PagId    int
	MsgBody  string
}

type retailUiReq struct {
	Req     string `json:"Req"`
	CmdMode string `json:"cmdMode"`
	ReqData string `json:"ReqData"`
	CmdData string `json:"cmdData"`
}

func getRetailProvider() *DetailRecProvider {
	retailProviderOnce.Do(func() {
		retailProvider = NewDetailRecProvider()
	})
	return retailProvider
}

func RetailInProcEnabled() bool {
	return true
}

func isScaleBoundToRetail(scaleId int64) bool {
	if mSrvMgr == nil {
		return true
	}
	found := false
	for _, rel := range mSrvMgr.srvScaleRel {
		if rel != nil && rel.ScaleId == scaleId && rel.SrvId == retailServiceId {
			found = true
			if rel.IsUsed {
				return true
			}
		}
	}
	if !found {
		return true
	}
	return false
}

func RetailInProcOnDetail(scaleId int64, list DetailList) {
	defer func() {
		if r := recover(); r != nil {
			l.Log.Errorf("RetailInProcOnDetail panic: %v", r)
		}
	}()

	if !isScaleBoundToRetail(scaleId) {
		return
	}
	p := getRetailProvider()
	if p == nil {
		return
	}

	total := list.Total
	total.RecId = 0
	if err := p.InsertTotalRec(total); err != nil {
		l.Log.Errorf("retail: insert total failed: %v", err)
		return
	}
	for _, rec := range list.Details {
		rec.RecId = 0
		if err := p.InsertDetailRec(rec); err != nil {
			l.Log.Errorf("retail: insert detail failed: %v", err)
		}
	}

	// 通知 UI 接收新交易记录
	retailSendToUi(respDetailAdd, "ok")
}

func RetailInProcHandleUiReq(jsonStr string) bool {
	defer func() {
		if r := recover(); r != nil {
			l.Log.Errorf("RetailInProcHandleUiReq panic: %v", r)
		}
	}()

	var req retailUiReq
	if err := json.UnmarshalFromString(jsonStr, &req); err != nil {
		l.Log.Errorf("retail: parse UI req failed: %v", err)
		return false
	}

	cmd := req.Req
	if cmd == "" {
		cmd = req.CmdMode
	}

	switch cmd {
	case "get_detail_list":
		lists := retailLoadAll()
		retailMu.Lock()
		retailLastPushedId = retailMaxId(lists)
		retailMu.Unlock()
		retailSendPaged(respDetailList, lists)
		return true

	case "get_new_detail":
		lists := retailLoadAll()
		retailMu.Lock()
		last := retailLastPushedId
		newer := make([]DetailList, 0)
		for _, item := range lists {
			if item.Total.RecId > last {
				newer = append(newer, item)
			}
		}
		if m := retailMaxId(lists); m > retailLastPushedId {
			retailLastPushedId = m
		}
		retailMu.Unlock()
		retailSendPaged(respNewDetail, newer)
		return true
	}

	return false
}

func RetailInProcServiceAction(serviceId int64) (string, bool) {
	if serviceId != retailServiceId {
		return "", false
	}
	return "Status:" + SRV_STATUS_STARTED, true
}

func retailLoadAll() []DetailList {
	p := getRetailProvider()
	if p == nil {
		return []DetailList{}
	}
	lists, err := p.GetRecsList()
	if err != nil {
		l.Log.Errorf("retail: load detail list failed: %v", err)
		return []DetailList{}
	}
	if lists == nil {
		lists = []DetailList{}
	}
	for i := range lists {
		if lists[i].Details == nil {
			lists[i].Details = []DetailRec{}
		}
	}
	return lists
}

func retailMaxId(lists []DetailList) uint {
	var m uint
	for _, item := range lists {
		if item.Total.RecId > m {
			m = item.Total.RecId
		}
	}
	return m
}

func retailSendPaged(msgType ScaleMgrRespMsgType, lists []DetailList) {
	if lists == nil {
		lists = []DetailList{}
	}
	for i := range lists {
		if lists[i].Details == nil {
			lists[i].Details = []DetailRec{}
		}
	}

	body, err := json.MarshalToString(lists)
	if err != nil {
		l.Log.Errorf("retail: json marshal failed: %v", err)
		body = "[]"
	}
	chunks := splitUtf8(body, retailPakMaxBytes)
	for i, c := range chunks {
		pak := retailPak{PakCount: len(chunks), PagId: i + 1, MsgBody: c}
		pakStr, err := json.MarshalToString(pak)
		if err != nil {
			l.Log.Errorf("retail: pak marshal failed: %v", err)
			return
		}
		retailSendToUi(msgType, pakStr)
	}
}

func retailSendToUi(msgType ScaleMgrRespMsgType, body string) {
	if mSrvMgr == nil || mSrvMgr.recvScaleMgrMsg == nil {
		return
	}
	mSrvMgr.recvScaleMgrMsg <- &ScaleMgrRespMsg{MsgType: msgType, MsgBody: body}
}

func splitUtf8(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}
	var out []string
	for len(s) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		if cut == 0 {
			cut = max
		}
		out = append(out, s[:cut])
		s = s[cut:]
	}
	if len(s) > 0 {
		out = append(out, s)
	}
	return out
}
