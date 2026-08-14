package versions

var SchemaList = SchemaListMap{
	"2004/08": SchemaListArray{
		Soap:       "http://www.w3.org/2003/05/soap-envelope",
		Addressing: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
		Discovery:  "http://schemas.xmlsoap.org/ws/2004/10/discovery",
		Mex:        "http://schemas.xmlsoap.org/ws/2004/09/mex",
		DevProf:    "http://schemas.xmlsoap.org/ws/2005/05/devprof/",
		PnPX:       "http://schemas.microsoft.com/windows/pnpx/2005/10",
		Pub:        "http://schemas.microsoft.com/windows/pub/2005/07",
	},
	"2005/04": SchemaListArray{
		Soap:       "http://www.w3.org/2003/05/soap-envelope",
		Addressing: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
		Discovery:  "http://schemas.xmlsoap.org/ws/2005/04/discovery",
		Mex:        "http://schemas.xmlsoap.org/ws/2004/09/mex",
		DevProf:    "http://schemas.xmlsoap.org/ws/2006/02/devprof",
		PnPX:       "http://schemas.microsoft.com/windows/pnpx/2005/10",
		Pub:        "http://schemas.microsoft.com/windows/pub/2005/07",
	},
	"2009/01": SchemaListArray{
		Soap:       "http://www.w3.org/2003/05/soap-envelope",
		Addressing: "http://www.w3.org/2005/08/addressing",
		Discovery:  "http://docs.oasis-open.org/ws-dd/ns/discovery/2009/01",
		Mex:        "http://www.w3.org/2009/09/ws-mex",
		DevProf:    "http://docs.oasis-open.org/ws-dd/ns/dpws/2009/01",
		PnPX:       "http://schemas.microsoft.com/windows/pnpx/2005/10",
		Pub:        "http://schemas.microsoft.com/windows/pub/2005/07",
	},
}

var ToValueList = map[string]map[string]string{
	"2004/08": {
		"request": "urn:schemas-xmlsoap-org:ws:2004:10:discovery",
		"reply":   SchemaList["2004/08"][Addressing] + "/role/anonymous",
	},
	"2005/04": {
		"request": "urn:schemas-xmlsoap-org:ws:2005:04:discovery",
		"reply":   SchemaList["2005/04"][Addressing] + "/role/anonymous",
	},
	"2009/01": {
		"request": "urn:docs-oasis-open-org:ws-dd:ns:discovery:2009:01",
		"reply":   SchemaList["2009/01"][Addressing] + "/anonymous",
	},
}

const TransferSchema = "http://schemas.xmlsoap.org/ws/2004/09/transfer"
