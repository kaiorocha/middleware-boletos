package repository

import (
	"errors"

	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
	"github.com/lib/pq"
)

func translatePostgresError(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		return err
	}

	switch pqErr.Constraint {
	case "idx_users_tenant_lower_email_unique":
		return service.NewDuplicateResource("Já existe um usuário com este e-mail neste tenant.")
	case "idx_customers_tenant_document_unique":
		return service.NewDuplicateResource("Já existe um cliente com este documento neste tenant.")
	case "idx_providers_tenant_lower_name_unique":
		return service.NewDuplicateResource("Já existe um provedor com este nome neste tenant.")
	case "idx_boletos_tenant_external_id_unique":
		return service.NewDuplicateResource("Já existe um boleto com este external_id neste tenant.")
	case "idx_boletos_tenant_our_number_unique":
		return service.NewDuplicateResource("Já existe um boleto com este nosso número neste tenant.")
	default:
		return service.NewDuplicateResource("Já existe um recurso com estes dados neste tenant.")
	}
}
