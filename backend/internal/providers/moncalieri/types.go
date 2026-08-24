package moncalieri

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
