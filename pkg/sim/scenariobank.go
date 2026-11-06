package sim

// ScenarioBank is the second game's scenario.dll DWORD bank: 1024 slots
// (R2-ENGINE-045). Slots ObjectiveSlotBase..+ObjectiveSlots-1 hold the
// mission's objective states; objective n is slot 752 + n (R2-ENGINE-046,
// R2-ENGINE-888). The functions below are the bank's one slot writer: the
// script's instants 35 and 36, a location entry and NewGame all go through
// them. A LOAD restores the bank raw and keeps every slot (R2-SESSION-031,
// R2-SESSION-348).
type ScenarioBank = [1024]int32

const (
	ObjectiveSlotBase = 752
	ObjectiveSlots    = 16
)

// The events instant 36 raises: 253 on a completion from a state without
// bit 2, 254 on every mode 4 (R2-ENGINE-046, R2-ENGINE-891).
const (
	ObjectiveEventDone   int32 = 253
	ObjectiveEventMode4  int32 = 254
	objectiveNoEvent     int32 = 0
	scenarioLocationSlot       = 768
)

// NewGameScenarioBank is the bank NewGame leaves: every slot cleared, then
// slot 768 = 10 (R2-ENGINE-045).
func NewGameScenarioBank() ScenarioBank {
	var bank ScenarioBank
	bank[scenarioLocationSlot] = 10
	return bank
}

// EnterLocation clears the objective slots, as every location entry does
// (R2-ENGINE-046, R2-ENGINE-896).
func EnterLocation(bank *ScenarioBank) {
	clear(bank[ObjectiveSlotBase : ObjectiveSlotBase+ObjectiveSlots])
}

// SetScenario is instant 35's SetVar: slot index takes value. An index
// outside the bank writes nothing (DIV-2354).
func SetScenario(bank *ScenarioBank, index int64, value int32) bool {
	if index < 0 || index >= int64(len(bank)) {
		return false
	}
	bank[index] = value
	return true
}

// WriteObjective is instant 36 (n, mode) on slot 752 + n and answers the
// event it raises, 0 for none (R2-ENGINE-046): mode 1 stores 1 over 0 only;
// mode 2 leaves 4 alone, else stores 3 and raises 253 when bit 2 was clear;
// mode 4 stores 5 and raises 254; any other mode is stored as given. A slot
// outside the bank writes nothing (DIV-2354).
func WriteObjective(bank *ScenarioBank, n int64, mode int32) int32 {
	index := int64(ObjectiveSlotBase) + n
	if index < 0 || index >= int64(len(bank)) {
		return objectiveNoEvent
	}
	old := bank[index]
	switch mode {
	case 1:
		if old == 0 {
			bank[index] = 1
		}
	case 2:
		if old == 4 {
			return objectiveNoEvent
		}
		bank[index] = 3
		if old&2 == 0 {
			return ObjectiveEventDone
		}
	case 4:
		bank[index] = 5
		return ObjectiveEventMode4
	default:
		bank[index] = mode
	}
	return objectiveNoEvent
}
