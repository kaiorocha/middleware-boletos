package moncalieri

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func mapIssueRequest(cfg Config, req types.IssueRequest) (envelope[gerarBoletoData], error) {
	if req.Payer == nil {
		return envelope[gerarBoletoData]{}, providererrors.New(errInvalidRequest, "payer data is required for Moncalieri boleto issuance", providerName, false)
	}
	payer, err := mapSacado(*req.Payer)
	if err != nil {
		return envelope[gerarBoletoData]{}, err
	}
	identifier := strings.TrimSpace(req.ExternalID)
	if identifier == "" {
		identifier = req.BoletoID
	}
	instructions := strings.TrimSpace(req.Instructions)
	if instructions == "" {
		instructions = strings.TrimSpace(cfg.Instrucoes)
	}
	if instructions == "" {
		instructions = "Nao receber apos o vencimento."
	}

	return envelope[gerarBoletoData]{
		Data: gerarBoletoData{
			CodigoCanal:          cfg.CodigoCanal,
			CodigoCliente:        cfg.CodigoCliente,
			DataVencimento:       req.DueDate.Format("2006-01-02"),
			Valor:                float64(req.AmountCents) / 100,
			Email:                strings.TrimSpace(req.Payer.Email),
			DadosSacado:          payer,
			IdentificadorCliente: identifier,
			RetornarBase64:       true,
			Instrucoes:           instructions,
		},
	}, nil
}

func mapSacado(payer types.Payer) (sacadoData, error) {
	document, err := parseRequiredDigits64(payer.Document, "payer document")
	if err != nil {
		return sacadoData{}, err
	}
	postalCode, err := parseRequiredDigits(payer.PostalCode, "payer postal code")
	if err != nil {
		return sacadoData{}, err
	}
	ddi, err := parseRequiredDigits(payer.CountryCode, "payer country code")
	if err != nil {
		return sacadoData{}, err
	}
	ddd, err := parseRequiredDigits(payer.AreaCode, "payer area code")
	if err != nil {
		return sacadoData{}, err
	}
	phone, err := parseRequiredDigits64(payer.PhoneNumber, "payer phone number")
	if err != nil {
		return sacadoData{}, err
	}
	out := sacadoData{
		CpfCnpj:               document,
		Nome:                  strings.TrimSpace(payer.Name),
		Endereco:              strings.TrimSpace(payer.Address),
		Bairro:                strings.TrimSpace(payer.District),
		Cidade:                strings.TrimSpace(payer.City),
		Cep:                   postalCode,
		Uf:                    strings.ToUpper(strings.TrimSpace(payer.State)),
		DdiTerceiro:           ddi,
		DddTerceiro:           ddd,
		NumeroCelularTerceiro: phone,
	}
	if out.Nome == "" || out.Endereco == "" || out.Bairro == "" || out.Cidade == "" || out.Uf == "" {
		return sacadoData{}, providererrors.New(errInvalidRequest, "payer name, address, district, city and state are required", providerName, false)
	}
	if len(out.Uf) != 2 {
		return sacadoData{}, providererrors.New(errInvalidRequest, "payer state must have 2 letters", providerName, false)
	}
	return out, nil
}

func mapIssueResponse(req types.IssueRequest, resp gerarBoletoResponse) (types.IssueResponse, error) {
	if err := responseError(resp.ResultCode, resp.Message, resp.ValidationData); err != nil {
		return types.IssueResponse{}, err
	}
	if resp.Data.NossoNumero == "" || resp.Data.LinhaDigitavel == "" || resp.Data.CodigoBarras == "" || resp.Data.Base64 == "" {
		return types.IssueResponse{}, providererrors.New(errProviderUnexpected, "provider response is missing boleto fields", providerName, false)
	}
	externalID := strings.TrimSpace(req.ExternalID)
	if externalID == "" {
		externalID = strings.TrimSpace(req.BoletoID)
	}
	if externalID == "" {
		externalID = resp.Data.NossoNumero
	}
	return types.IssueResponse{
		ExternalID:    externalID,
		Barcode:       resp.Data.CodigoBarras,
		DigitableLine: resp.Data.LinhaDigitavel,
		OurNumber:     resp.Data.NossoNumero,
		Base64:        resp.Data.Base64,
		Status:        types.StatusIssued,
		IssuedAt:      time.Now().UTC(),
	}, nil
}

func mapBoletoSummary(data consultarBoletoResponseData) types.BoletoSummary {
	return types.BoletoSummary{
		ExternalID:    data.IdentificadorCliente,
		OurNumber:     data.NossoNumero,
		Status:        MapStatus(data.Status),
		AmountCents:   data.Valor,
		DueDate:       parseProviderDate(data.DataVencimento),
		Barcode:       data.CodigoBarras,
		DigitableLine: data.LinhaDigitavel,
	}
}

func MapStatus(status string) types.BoletoStatus {
	normalized := normalizeText(status)
	switch normalized {
	case "pago", "pagos", "liquidado", "liquidados":
		return types.StatusPaid
	case "pendente", "pendentes", "registrado", "registrados":
		return types.StatusIssued
	case "baixado", "baixados", "cancelado", "cancelados":
		return types.StatusCancelled
	case "vencido", "vencidos":
		return types.StatusExpired
	case "falha", "falhado", "erro", "rejeitado", "rejeitados":
		return types.StatusFailed
	default:
		if normalized == "" {
			return types.StatusProcessing
		}
		return types.StatusProcessing
	}
}

func responseError(resultCode int, message string, validation validationData) error {
	if resultCode == 0 && len(validation.Errors) == 0 {
		return nil
	}
	if len(validation.Errors) > 0 {
		first := validation.Errors[0]
		return providererrors.New(errProviderValidation, fmt.Sprintf("%s: %s", first.FieldName, first.ErrorMessage), providerName, false)
	}
	if strings.TrimSpace(message) != "" {
		return providererrors.New(errProviderValidation, message, providerName, false)
	}
	return providererrors.New(errProviderUnexpected, fmt.Sprintf("provider returned result code %d", resultCode), providerName, false)
}

func parseProviderDate(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"20060102", "2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseRequiredDigits(value, field string) (int, error) {
	n, err := parseRequiredDigits64(value, field)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func parseRequiredDigits64(value, field string) (int64, error) {
	digits := onlyDigits(value)
	if digits == "" {
		return 0, providererrors.New(errInvalidRequest, field+" is required", providerName, false)
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, providererrors.New(errInvalidRequest, field+" must contain only valid digits", providerName, false)
	}
	return n, nil
}

func onlyDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c")
	return replacer.Replace(value)
}
