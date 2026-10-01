package moncalieri

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	CodigoCanal    int    `json:"codigo_canal"`
	CodigoCliente  int    `json:"codigo_cliente"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Instrucoes     string `json:"instrucoes"`
}

type envelope[T any] struct {
	Data T `json:"Data"`
}

type gerarBoletoData struct {
	CodigoCanal          int        `json:"CodigoCanal"`
	CodigoCliente        int        `json:"CodigoCliente"`
	DataVencimento       string     `json:"DataVencimento"`
	Valor                float64    `json:"Valor"`
	Email                string     `json:"Email,omitempty"`
	DadosSacado          sacadoData `json:"DadosSacado"`
	IdentificadorCliente string     `json:"IdentificadorCliente"`
	RetornarBase64       bool       `json:"RetornarBase64"`
	Instrucoes           string     `json:"Instrucoes"`
}

type sacadoData struct {
	CpfCnpj               int64  `json:"CpfCnpj"`
	Nome                  string `json:"Nome"`
	Endereco              string `json:"Endereco"`
	Bairro                string `json:"Bairro"`
	Cidade                string `json:"Cidade"`
	Cep                   int    `json:"Cep"`
	Uf                    string `json:"Uf"`
	DdiTerceiro           int    `json:"DdiTerceiro"`
	DddTerceiro           int    `json:"DddTerceiro"`
	NumeroCelularTerceiro int64  `json:"NumeroCelularTerceiro"`
}

type gerarBoletoResponse struct {
	Data           gerarBoletoResponseData `json:"Data"`
	ResultCode     int                     `json:"ResultCode"`
	Message        string                  `json:"Message"`
	ValidationData validationData          `json:"ValidationData"`
	rawBody        []byte
}

type gerarBoletoResponseData struct {
	NossoNumero     string `json:"NossoNumero"`
	LinhaDigitavel  string `json:"LinhaDigitavel"`
	CodigoBarras    string `json:"CodigoBarras"`
	Base64          string `json:"Base64"`
	BoletoBase64    string `json:"BoletoBase64"`
	ArquivoBase64   string `json:"ArquivoBase64"`
	PdfBase64       string `json:"PdfBase64"`
	DocumentoBase64 string `json:"DocumentoBase64"`
}

func (r *gerarBoletoResponse) captureResponseBody(body []byte) {
	r.rawBody = append(r.rawBody[:0], body...)
}

func (d gerarBoletoResponseData) boletoBase64() string {
	for _, value := range []string{d.Base64, d.BoletoBase64, d.ArquivoBase64, d.PdfBase64, d.DocumentoBase64} {
		if value != "" {
			return value
		}
	}
	return ""
}

type consultarBoletoData struct {
	CodigoCanal    int    `json:"CodigoCanal"`
	CodigoCliente  int    `json:"CodigoCliente"`
	NossoNumero    string `json:"NossoNumero"`
	RetornarBase64 bool   `json:"RetornarBase64,omitempty"`
}

type consultarBoletoResponse struct {
	Data           consultarBoletoResponseData `json:"Data"`
	ResultCode     int                         `json:"ResultCode"`
	Message        string                      `json:"Message"`
	ValidationData validationData              `json:"ValidationData"`
	rawBody        []byte
}

func (r *consultarBoletoResponse) captureResponseBody(body []byte) {
	r.rawBody = append(r.rawBody[:0], body...)
}

func (r *consultarBoletoResponse) UnmarshalJSON(body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '"' {
		var encoded string
		if err := json.Unmarshal(body, &encoded); err != nil {
			return err
		}
		return r.UnmarshalJSON([]byte(encoded))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	data := rawField(raw, "Data")
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &r.Data); err != nil {
			var list []consultarBoletoResponseData
			if listErr := json.Unmarshal(data, &list); listErr != nil || len(list) == 0 {
				return err
			}
			r.Data = list[0]
		}
	}
	r.ResultCode = parseRawInt(rawField(raw, "ResultCode"))
	r.Message = parseRawString(rawField(raw, "Message"))
	if validation := rawField(raw, "ValidationData"); len(validation) > 0 && string(validation) != "null" {
		_ = json.Unmarshal(validation, &r.ValidationData)
	}
	return nil
}

type consultarBoletoLoteData struct {
	CodigoCanal    int    `json:"CodigoCanal"`
	CodigoCliente  int    `json:"CodigoCliente"`
	DataInicial    string `json:"DataInicial"`
	DataFinal      string `json:"DataFinal"`
	TipoPesquisa   string `json:"TipoPesquisa"`
	StatusPesquisa string `json:"StatusPesquisa"`
}

type consultarBoletoLoteResponse struct {
	Data           []consultarBoletoResponseData `json:"Data"`
	ResultCode     int                           `json:"ResultCode"`
	Message        string                        `json:"Message"`
	ValidationData validationData                `json:"ValidationData"`
}

type consultarBoletoResponseData struct {
	Status               string `json:"Status"`
	Valor                int64  `json:"Valor"`
	ValorPago            int64  `json:"ValorPago"`
	DataVencimento       string `json:"DataVencimento"`
	NossoNumero          string `json:"NossoNumero"`
	LinhaDigitavel       string `json:"LinhaDigitavel"`
	CodigoBarras         string `json:"CodigoBarras"`
	IdentificadorCliente string `json:"IdentificadorCliente"`
	Base64               string `json:"Base64"`
	BoletoBase64         string `json:"BoletoBase64"`
	ArquivoBase64        string `json:"ArquivoBase64"`
	PdfBase64            string `json:"PdfBase64"`
}

func (d *consultarBoletoResponseData) UnmarshalJSON(body []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	d.Status = parseRawString(rawField(raw, "Status"))
	d.Valor = parseRawInt64(rawField(raw, "Valor"))
	d.ValorPago = parseRawInt64(rawField(raw, "ValorPago"))
	d.DataVencimento = parseRawString(rawField(raw, "DataVencimento"))
	d.NossoNumero = parseRawString(rawField(raw, "NossoNumero"))
	d.LinhaDigitavel = parseRawString(rawField(raw, "LinhaDigitavel"))
	d.CodigoBarras = parseRawString(rawField(raw, "CodigoBarras"))
	d.IdentificadorCliente = parseRawString(rawField(raw, "IdentificadorCliente"))
	d.Base64 = parseRawString(rawField(raw, "Base64"))
	d.BoletoBase64 = parseRawString(rawField(raw, "BoletoBase64"))
	d.ArquivoBase64 = parseRawString(rawField(raw, "ArquivoBase64"))
	d.PdfBase64 = parseRawString(rawField(raw, "PdfBase64"))
	return nil
}

func rawField(raw map[string]json.RawMessage, name string) json.RawMessage {
	for key, value := range raw {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return nil
}
func parseRawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	var generic any
	if json.Unmarshal(raw, &generic) == nil {
		return strings.TrimSpace(fmt.Sprint(generic))
	}
	return ""
}
func parseRawInt(raw json.RawMessage) int { return int(parseRawInt64(raw)) }
func parseRawInt64(raw json.RawMessage) int64 {
	value := strings.TrimSpace(parseRawString(raw))
	if value == "" {
		value = strings.Trim(string(raw), `"`)
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	parsed, _ := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	return int64(parsed)
}

func (d consultarBoletoResponseData) boletoBase64() string {
	for _, value := range []string{d.Base64, d.BoletoBase64, d.ArquivoBase64, d.PdfBase64} {
		if value != "" {
			return value
		}
	}
	return ""
}

type solicitarBaixaBoletoResponse struct {
	Data           string         `json:"Data"`
	ResultCode     int            `json:"ResultCode"`
	Message        string         `json:"Message"`
	ValidationData validationData `json:"ValidationData"`
}

type validationData struct {
	ResultCode int               `json:"ResultCode"`
	Message    string            `json:"Message"`
	Errors     []validationError `json:"Errors"`
}

type validationError struct {
	ErrorMessage string `json:"ErrorMessage"`
	FieldName    string `json:"FieldName"`
}
