package core

import (
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

var pdfiumPool pdfium.Pool
var pdfiumInst pdfium.Pdfium

func InitPDFium() error {
	var err error
	pdfiumPool, err = webassembly.Init(webassembly.Config{
		MinIdle: 1, MaxIdle: 2, MaxTotal: 4,
	})
	if err != nil {
		return fmt.Errorf("pdfium init: %w", err)
	}
	pdfiumInst, err = pdfiumPool.GetInstance(30 * time.Second)
	if err != nil {
		return fmt.Errorf("pdfium instance: %w", err)
	}
	return nil
}

func ClosePDFium() {
	if pdfiumPool != nil {
		pdfiumPool.Close()
	}
}

func PDFToImages(path string, outDir string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	doc, err := pdfiumInst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer pdfiumInst.FPDF_CloseDocument(
		&requests.FPDF_CloseDocument{Document: doc.Document})
	pageCount, err := pdfiumInst.FPDF_GetPageCount(
		&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return nil, fmt.Errorf("pagecount: %w", err)
	}
	var outs []string
	for i := 0; i < pageCount.PageCount; i++ {
		r, err := pdfiumInst.RenderPageInDPI(&requests.RenderPageInDPI{
			DPI: 200,
			Page: requests.Page{ByIndex: &requests.PageByIndex{
				Document: doc.Document, Index: i,
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("render %d: %w", i, err)
		}
		out := fmt.Sprintf("%s/page_%d.png", outDir, i+1)
		f, err := os.Create(out)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", out, err)
		}
		if err := png.Encode(f, r.Result.Image); err != nil {
			f.Close()
			return nil, fmt.Errorf("encode %s: %w", out, err)
		}
		f.Close()
		outs = append(outs, out)
	}
	return outs, nil
}
