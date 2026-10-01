-- Historical PAID rows deliberately retain NULL: their confirmation date is unknown.
ALTER TABLE boletos ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

CREATE OR REPLACE FUNCTION track_boleto_payment_confirmation() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status = 'PAID' AND NEW.paid_at IS NULL THEN
            NEW.paid_at := now();
        END IF;
    ELSIF NEW.status = 'PAID' AND OLD.status IS DISTINCT FROM 'PAID' AND OLD.paid_at IS NULL THEN
        NEW.paid_at := now();
    ELSE
        NEW.paid_at := OLD.paid_at;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS boleto_payment_confirmation ON boletos;
CREATE TRIGGER boleto_payment_confirmation
BEFORE INSERT OR UPDATE ON boletos
FOR EACH ROW EXECUTE FUNCTION track_boleto_payment_confirmation();
