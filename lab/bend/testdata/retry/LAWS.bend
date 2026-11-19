import Base
import ./policy.bend as Policy

law charge_stops_on_unknown:
  {Policy.may_retry_charge(Policy.Unknown{}) == False{} : Bool}

law charge_can_retry_when_not_applied:
  {Policy.may_retry_charge(Policy.NotApplied{}) == True{} : Bool}

law charge_stops_on_success:
  {Policy.may_retry_charge(Policy.Applied{}) == False{} : Bool}
