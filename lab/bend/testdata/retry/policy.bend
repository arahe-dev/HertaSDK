import Base

type Attempt is Data:
  NotApplied{}
  Unknown{}
  Applied{}

def may_retry_charge(result: Attempt) -> Bool:
  match result:
    case NotApplied{}:
      True{}
    case Unknown{}:
      False{}
    case Applied{}:
      False{}
