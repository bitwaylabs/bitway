package types

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"slices"

	btcschnorr "github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	errorsmod "cosmossdk.io/errors"

	"github.com/bitwaylabs/bitway/bitcoin/crypto/adaptor"
	"github.com/bitwaylabs/bitway/bitcoin/crypto/schnorr"
	btcbridgetypes "github.com/bitwaylabs/bitway/x/btcbridge/types"
	dlctypes "github.com/bitwaylabs/bitway/x/dlc/types"
)

const (
	// liquidation cet sig hash type for borrower
	BorrowerLiquidationCetSigHashType = txscript.SigHashNone | txscript.SigHashAnyOneCanPay

	// liquidation cet sig hash type for DCM
	DCMLiquidationCetSigHashType = txscript.SigHashDefault

	// default sig hash type used for repayment or redemption
	DefaultSigHashType = txscript.SigHashDefault | txscript.SigHashAnyOneCanPay
)

const (
	// outcome index for liquidated
	LiquidatedOutcomeIndex = 0

	// outcome index for default liquidated
	DefaultLiquidatedOutcomeIndex = 1

	// outcome index for repaid
	RepaidOutcomeIndex = 2
)

// BuildDLCMeta builds the dlc meta from the given params
// Assume that the given params are valid
func BuildDLCMeta(borrowerPubKey string, borrowerAuthPubKey string, dcmPubKey string, finalTimeout int64) (*DLCMeta, error) {
	borrowerPubKeyBytes, _ := hex.DecodeString(borrowerPubKey)
	borrowerAuthPubKeyBytes, _ := hex.DecodeString(borrowerAuthPubKey)
	dcmPubKeyBytes, _ := hex.DecodeString(dcmPubKey)

	internalKey := GetInternalKey(borrowerPubKeyBytes, dcmPubKeyBytes)

	liquidationScript, repaymentScript, timeoutRefundScript, _ := GetVaultScripts(borrowerPubKeyBytes, borrowerAuthPubKeyBytes, dcmPubKeyBytes, finalTimeout)

	tapscriptTree := GetTapscriptTree([][]byte{liquidationScript, repaymentScript, timeoutRefundScript})

	liquidationScriptControlBlock, err := GetControlBlock(tapscriptTree, liquidationScript, internalKey)
	if err != nil {
		return nil, err
	}

	repaymentScriptControlBlock, err := GetControlBlock(tapscriptTree, repaymentScript, internalKey)
	if err != nil {
		return nil, err
	}

	timeoutRefundScriptControlBlock, err := GetControlBlock(tapscriptTree, timeoutRefundScript, internalKey)
	if err != nil {
		return nil, err
	}

	return &DLCMeta{
		InternalKey:         hex.EncodeToString(btcschnorr.SerializePubKey(internalKey)),
		LiquidationScript:   GetLeafScript(liquidationScript, liquidationScriptControlBlock),
		RepaymentScript:     GetLeafScript(repaymentScript, repaymentScriptControlBlock),
		TimeoutRefundScript: GetLeafScript(timeoutRefundScript, timeoutRefundScriptControlBlock),
	}, nil
}

// VerifyCets verifies the given cets
func VerifyCets(dlcMeta *DLCMeta, depositTxs []*psbt.Packet, vaultPkScript []byte, borrowerPubKey string, borrowerAuthPubKey string, dcmPubKey string, dlcEvent *dlctypes.DLCEvent, liquidationCet string, liquidationAdaptorSignatures []string, defaultLiquidationAdaptorSignatures []string, repaymentCet string, repaymentSignatures []string) error {
	liquidationAdaptorPoint, err := dlctypes.GetSignaturePointFromEvent(dlcEvent, LiquidatedOutcomeIndex)
	if err != nil {
		return err
	}

	defaultLiquidationAdaptorPoint, err := dlctypes.GetSignaturePointFromEvent(dlcEvent, DefaultLiquidatedOutcomeIndex)
	if err != nil {
		return err
	}

	if err := VerifyLiquidationCet(dlcMeta, depositTxs, vaultPkScript, borrowerAuthPubKey, liquidationCet, liquidationAdaptorSignatures, liquidationAdaptorPoint); err != nil {
		return errorsmod.Wrapf(ErrInvalidCET, "invalid liquidation cet: %v", err)
	}

	if err := VerifyLiquidationCet(dlcMeta, depositTxs, vaultPkScript, borrowerAuthPubKey, liquidationCet, defaultLiquidationAdaptorSignatures, defaultLiquidationAdaptorPoint); err != nil {
		return errorsmod.Wrapf(ErrInvalidCET, "invalid default liquidation cet: %v", err)
	}

	if err := VerifyRepaymentCet(dlcMeta, depositTxs, vaultPkScript, borrowerPubKey, repaymentCet, repaymentSignatures); err != nil {
		return errorsmod.Wrapf(ErrInvalidCET, "invalid repayment cet: %v", err)
	}

	return nil
}

// VerifyLiquidationCet verifies the given liquidation cet and corresponding adaptor signatures
func VerifyLiquidationCet(dlcMeta *DLCMeta, depositTxs []*psbt.Packet, vaultPkScript []byte, borrowerAuthPubKey string, liquidationCET string, adaptorSignatures []string, adaptorPoint []byte) error {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(liquidationCET)), true)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidCET, "failed to deserialize cet")
	}

	// no output is allowed; output will be populated later
	if len(p.UnsignedTx.TxOut) != 0 {
		return errorsmod.Wrap(ErrInvalidCET, "incorrect tx out")
	}

	vaultUtxos, err := GetVaultUtxos(depositTxs, vaultPkScript)
	if err != nil {
		return err
	}

	if len(p.UnsignedTx.TxIn) != len(vaultUtxos) {
		return errorsmod.Wrap(ErrInvalidCET, "incorrect input number")
	}

	for i, txIn := range p.UnsignedTx.TxIn {
		if txIn.PreviousOutPoint.Hash.String() != vaultUtxos[i].Txid {
			return errorsmod.Wrap(ErrInvalidCET, "incorrect previous tx hash")
		}

		if txIn.PreviousOutPoint.Index != uint32(vaultUtxos[i].Vout) {
			return errorsmod.Wrap(ErrInvalidCET, "incorrect previous tx out index")
		}

		if p.Inputs[i].WitnessUtxo == nil {
			return errorsmod.Wrap(ErrInvalidCET, "missing witness utxo")
		}

		if !bytes.Equal(p.Inputs[i].WitnessUtxo.PkScript, vaultPkScript) {
			return errorsmod.Wrap(ErrInvalidCET, "mismatched witness utxo pk script")
		}

		if p.Inputs[i].WitnessUtxo.Value != int64(vaultUtxos[i].Amount) {
			return errorsmod.Wrap(ErrInvalidCET, "mismatched witness utxo value")
		}
	}

	if len(adaptorSignatures) != len(p.Inputs) {
		return errorsmod.Wrap(ErrInvalidAdaptorSignatures, "incorrect signature number")
	}

	borrowerAuthPubKeyBytes, err := hex.DecodeString(borrowerAuthPubKey)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidPubKey, "failed to decode borrower auth public key")
	}

	script, _, err := UnwrapLeafScript(dlcMeta.LiquidationScript)
	if err != nil {
		return err
	}

	for i, signature := range adaptorSignatures {
		sigHash, err := CalcTapscriptSigHash(p, i, BorrowerLiquidationCetSigHashType, script)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to calculate sig hash")
		}

		sigBytes, err := hex.DecodeString(signature)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidAdaptorSignature, "failed to decode adaptor signature")
		}

		if !adaptor.Verify(sigBytes, sigHash, borrowerAuthPubKeyBytes, adaptorPoint) {
			return ErrInvalidAdaptorSignature
		}
	}

	return nil
}

// VerifyRepaymentCet verifies the given repayment cet and corresponding signatures
func VerifyRepaymentCet(dlcMeta *DLCMeta, depositTxs []*psbt.Packet, vaultPkScript []byte, borrowerPubKey string, repaymentCet string, signatures []string) error {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(repaymentCet)), true)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidCET, "failed to deserialize cet")
	}

	if len(p.UnsignedTx.TxOut) == 0 {
		return errorsmod.Wrap(ErrInvalidCET, "missing tx out")
	}

	if slices.ContainsFunc(p.UnsignedTx.TxOut, btcbridgetypes.IsDustOut) {
		return errorsmod.Wrap(ErrInvalidCET, "dust tx out")
	}

	script, controlBlock, err := UnwrapLeafScript(dlcMeta.RepaymentScript)
	if err != nil {
		return err
	}

	witnessSize := getCetWitnessSize(CetType_REPAYMENT, script, controlBlock)

	fee, err := p.GetTxFee()
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidCET, "failed to get tx fee: %v", err)
	}

	if int64(fee) < GetTxVirtualSize(p.UnsignedTx, witnessSize) {
		return errorsmod.Wrap(ErrInvalidCET, "too low fee rate")
	}

	if err := CheckTransactionWeight(p.UnsignedTx, witnessSize); err != nil {
		return err
	}

	vaultUtxos, err := GetVaultUtxos(depositTxs, vaultPkScript)
	if err != nil {
		return err
	}

	if len(p.UnsignedTx.TxIn) != len(vaultUtxos) {
		return errorsmod.Wrap(ErrInvalidCET, "incorrect input number")
	}

	for i, txIn := range p.UnsignedTx.TxIn {
		if txIn.PreviousOutPoint.Hash.String() != vaultUtxos[i].Txid {
			return errorsmod.Wrap(ErrInvalidCET, "incorrect previous tx hash")
		}

		if txIn.PreviousOutPoint.Index != uint32(vaultUtxos[i].Vout) {
			return errorsmod.Wrap(ErrInvalidCET, "incorrect previous tx out index")
		}

		if p.Inputs[i].WitnessUtxo == nil {
			return errorsmod.Wrap(ErrInvalidCET, "missing witness utxo")
		}

		if !bytes.Equal(p.Inputs[i].WitnessUtxo.PkScript, vaultPkScript) {
			return errorsmod.Wrap(ErrInvalidCET, "mismatched witness utxo pk script")
		}

		if p.Inputs[i].WitnessUtxo.Value != int64(vaultUtxos[i].Amount) {
			return errorsmod.Wrap(ErrInvalidCET, "mismatched witness utxo value")
		}
	}

	if len(signatures) != len(p.Inputs) {
		return errorsmod.Wrap(ErrInvalidSignatures, "incorrect signature number")
	}

	borrowerPubKeyBytes, err := hex.DecodeString(borrowerPubKey)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidPubKey, "failed to decode borrower public key")
	}

	for i, signature := range signatures {
		sigHash, err := CalcTapscriptSigHash(p, i, DefaultSigHashType, script)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to calculate sig hash")
		}

		sigBytes, err := hex.DecodeString(signature)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidSignature, "failed to decode signature")
		}

		if !schnorr.Verify(sigBytes, sigHash, borrowerPubKeyBytes) {
			return ErrInvalidSignature
		}
	}

	return nil
}

// CreateLiquidationCET creates the liquidation cet
func CreateLiquidationCET(depositTxs []*psbt.Packet, vaultPkScript []byte, dcmPkScript []byte, internalKeyBytes []byte, leafScript LeafScript, feeRate int64) (string, error) {
	vaultUtxos, err := GetVaultUtxos(depositTxs, vaultPkScript)
	if err != nil {
		return "", err
	}

	script, controlBlock, err := UnwrapLeafScript(leafScript)
	if err != nil {
		return "", err
	}

	p, err := BuildPsbt(vaultUtxos, dcmPkScript, feeRate, getCetWitnessSize(CetType_LIQUIDATION, script, controlBlock))
	if err != nil {
		return "", err
	}

	for i := range p.Inputs {
		p.Inputs[i].TaprootInternalKey = internalKeyBytes
		p.Inputs[i].TaprootLeafScript = []*psbt.TaprootTapLeafScript{
			{
				ControlBlock: controlBlock,
				Script:       script,
				LeafVersion:  txscript.BaseLeafVersion,
			},
		}
	}

	psbtB64, err := p.B64Encode()
	if err != nil {
		return "", err
	}

	return psbtB64, nil
}

// CreateRepaymentCet creates the repayment cet
func CreateRepaymentCet(depositTxs []*psbt.Packet, vaultPkScript []byte, borrowerPkScript []byte, internalKeyBytes []byte, leafScript LeafScript, feeRate int64) (string, error) {
	vaultUtxos, err := GetVaultUtxos(depositTxs, vaultPkScript)
	if err != nil {
		return "", err
	}

	script, controlBlock, err := UnwrapLeafScript(leafScript)
	if err != nil {
		return "", err
	}

	p, err := BuildPsbt(vaultUtxos, borrowerPkScript, feeRate, getCetWitnessSize(CetType_REPAYMENT, script, controlBlock))
	if err != nil {
		return "", err
	}

	for i := range p.Inputs {
		p.Inputs[i].TaprootInternalKey = internalKeyBytes
		p.Inputs[i].TaprootLeafScript = []*psbt.TaprootTapLeafScript{
			{
				ControlBlock: controlBlock,
				Script:       script,
				LeafVersion:  txscript.BaseLeafVersion,
			},
		}
	}

	psbtB64, err := p.B64Encode()
	if err != nil {
		return "", err
	}

	return psbtB64, nil
}

// CreateTimeoutRefundTransaction creates the timeout refund tx
func CreateTimeoutRefundTransaction(depositTxs []*psbt.Packet, vaultPkScript []byte, borrowerPkScript []byte, internalKeyBytes []byte, leafScript LeafScript, feeRate int64) (string, error) {
	vaultUtxos, err := GetVaultUtxos(depositTxs, vaultPkScript)
	if err != nil {
		return "", err
	}

	script, controlBlock, err := UnwrapLeafScript(leafScript)
	if err != nil {
		return "", err
	}

	p, err := BuildPsbt(vaultUtxos, borrowerPkScript, feeRate, 64+len(script)+len(controlBlock))
	if err != nil {
		return "", err
	}

	for i := range p.Inputs {
		p.Inputs[i].SighashType = txscript.SigHashDefault
		p.Inputs[i].TaprootInternalKey = internalKeyBytes
		p.Inputs[i].TaprootLeafScript = []*psbt.TaprootTapLeafScript{
			{
				ControlBlock: controlBlock,
				Script:       script,
				LeafVersion:  txscript.BaseLeafVersion,
			},
		}
	}

	psbtB64, err := p.B64Encode()
	if err != nil {
		return "", err
	}

	return psbtB64, nil
}

// BuildSignedCet builds the signed cet from the given signatures
// Assume that the cet is valid and signatures match
func BuildSignedCet(cet string, borrowerPubKey string, borrowerSignatures []string, dcmPubKey string, dcmSignatures []string, cetType CetType) ([]byte, *chainhash.Hash, error) {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(cet)), true)
	if err != nil {
		return nil, nil, err
	}

	borrowerPubKeyBytes, err := hex.DecodeString(borrowerPubKey)
	if err != nil {
		return nil, nil, err
	}

	dcmPubKeyBytes, err := hex.DecodeString(dcmPubKey)
	if err != nil {
		return nil, nil, err
	}

	borrowerSigHashType, dcmSigHashType := GetCetSigHashTypes(cetType)

	for i, input := range p.Inputs {
		borrowerSig, err := hex.DecodeString(borrowerSignatures[i])
		if err != nil {
			return nil, nil, err
		}

		dcmSig, err := hex.DecodeString(dcmSignatures[i])
		if err != nil {
			return nil, nil, err
		}

		leafHash := txscript.NewBaseTapLeaf(input.TaprootLeafScript[0].Script).TapHash()

		p.Inputs[i].TaprootScriptSpendSig = []*psbt.TaprootScriptSpendSig{
			{
				XOnlyPubKey: dcmPubKeyBytes,
				LeafHash:    leafHash[:],
				Signature:   dcmSig,
				SigHash:     dcmSigHashType,
			},
			{
				XOnlyPubKey: borrowerPubKeyBytes,
				LeafHash:    leafHash[:],
				Signature:   borrowerSig,
				SigHash:     borrowerSigHashType,
			},
		}
	}

	if err := psbt.MaybeFinalizeAll(p); err != nil {
		return nil, nil, err
	}

	signedTx, err := psbt.Extract(p)
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	if err := signedTx.Serialize(&buf); err != nil {
		return nil, nil, err
	}

	txHash := signedTx.TxHash()

	return buf.Bytes(), &txHash, nil
}

// GetCetSigHashTypes gets the cet sig hash types for borrower and DCM by the given cet type
func GetCetSigHashTypes(cetType CetType) (txscript.SigHashType, txscript.SigHashType) {
	switch cetType {
	case CetType_LIQUIDATION, CetType_DEFAULT_LIQUIDATION:
		return BorrowerLiquidationCetSigHashType, DCMLiquidationCetSigHashType

	default:
		return DefaultSigHashType, DefaultSigHashType
	}
}

// GetCet gets the liquidation cet and corresponding type according to the given loan status
func GetLiquidationCetAndType(dlcMeta *DLCMeta, loanStatus LoanStatus) (LiquidationCet, CetType) {
	switch loanStatus {
	case LoanStatus_Liquidated:
		return dlcMeta.LiquidationCet, CetType_LIQUIDATION

	default:
		return dlcMeta.DefaultLiquidationCet, CetType_DEFAULT_LIQUIDATION
	}
}

// UpdateLiquidationCet updates the liquidation cet by the given type
func UpdateLiquidationCet(dlcMeta *DLCMeta, cetType CetType, cet LiquidationCet) {
	switch cetType {
	case CetType_LIQUIDATION:
		dlcMeta.LiquidationCet = cet

	default:
		dlcMeta.DefaultLiquidationCet = cet
	}
}

// UpdateLiquidationCetOutput adds the DCM output to the given cet
func AddDCMOutputToLiquidationCet(cet *psbt.Packet, script []byte, controlBlock []byte, dcmPkScript []byte, feeRate int64) error {
	// get total input amount
	inputAmount := int64(0)
	for _, input := range cet.Inputs {
		inputAmount += input.WitnessUtxo.Value
	}

	// add dcm output
	cet.UnsignedTx.TxOut[0] = wire.NewTxOut(0, dcmPkScript)

	// calculate tx fee
	witnessSize := getCetWitnessSize(CetType_LIQUIDATION, script, controlBlock)
	fee := GetTxVirtualSize(cet.UnsignedTx, witnessSize) * feeRate

	// update output value
	cet.UnsignedTx.TxOut[0].Value = inputAmount - fee

	if btcbridgetypes.IsDustOut(cet.UnsignedTx.TxOut[0]) {
		return ErrDustOutput
	}

	return CheckTransactionWeight(cet.UnsignedTx, witnessSize)
}

// GetCetInfo gets the cet info from the given event and script
func GetCetInfo(event *dlctypes.DLCEvent, outcomeIndex int, script []byte, controlBlock []byte, sigHashType txscript.SigHashType) (*CetInfo, error) {
	if event == nil {
		return nil, nil
	}

	signaturePoint, err := dlctypes.GetSignaturePointFromEvent(event, outcomeIndex)
	if err != nil {
		return nil, err
	}

	return &CetInfo{
		EventId:        event.Id,
		OutcomeIndex:   uint32(outcomeIndex),
		SignaturePoint: hex.EncodeToString(signaturePoint),
		Script:         GetLeafScript(script, controlBlock),
		SighashType:    uint32(sigHashType),
	}, nil
}

// GetLiquidationCetSigHashes gets the sig hashes of the liquidation cet
func GetLiquidationCetSigHashes(dlcMeta *DLCMeta) ([]string, error) {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(dlcMeta.LiquidationCet.Tx)), true)
	if err != nil {
		return nil, err
	}

	script, err := hex.DecodeString(dlcMeta.LiquidationScript.Script)
	if err != nil {
		return nil, err
	}

	sigHashes := []string{}

	for i := range p.Inputs {
		sigHash, err := CalcTapscriptSigHash(p, i, DCMLiquidationCetSigHashType, script)
		if err != nil {
			return nil, err
		}

		sigHashes = append(sigHashes, base64.StdEncoding.EncodeToString(sigHash))
	}

	return sigHashes, nil
}

// GetDefaultLiquidationCetSigHashes gets the sig hashes of the default liquidation cet
func GetDefaultLiquidationCetSigHashes(dlcMeta *DLCMeta) ([]string, error) {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(dlcMeta.DefaultLiquidationCet.Tx)), true)
	if err != nil {
		return nil, err
	}

	script, err := hex.DecodeString(dlcMeta.LiquidationScript.Script)
	if err != nil {
		return nil, err
	}

	sigHashes := []string{}

	for i := range p.Inputs {
		sigHash, err := CalcTapscriptSigHash(p, i, DCMLiquidationCetSigHashType, script)
		if err != nil {
			return nil, err
		}

		sigHashes = append(sigHashes, base64.StdEncoding.EncodeToString(sigHash))
	}

	return sigHashes, nil
}

// GetRepaymentCetSigHashes gets the sig hashes of the repayment cet
func GetRepaymentCetSigHashes(dlcMeta *DLCMeta) ([]string, error) {
	p, err := psbt.NewFromRawBytes(bytes.NewReader([]byte(dlcMeta.RepaymentCet.Tx)), true)
	if err != nil {
		return nil, err
	}

	script, err := hex.DecodeString(dlcMeta.RepaymentScript.Script)
	if err != nil {
		return nil, err
	}

	sigHashes := []string{}

	for i := range p.Inputs {
		sigHash, err := CalcTapscriptSigHash(p, i, DefaultSigHashType, script)
		if err != nil {
			return nil, err
		}

		sigHashes = append(sigHashes, base64.StdEncoding.EncodeToString(sigHash))
	}

	return sigHashes, nil
}

// GetLiquidationCetOutput gets the output value for the given liquidation cet
// Assume that the given cet is valid
func GetLiquidationCetOutput(liquidationCet string) int64 {
	p, _ := psbt.NewFromRawBytes(bytes.NewReader([]byte(liquidationCet)), true)

	return p.UnsignedTx.TxOut[0].Value
}

// GetLeafScript gets a leaf script from the given script and control block
func GetLeafScript(script []byte, controlBlock []byte) LeafScript {
	return LeafScript{
		Script:       hex.EncodeToString(script),
		ControlBlock: hex.EncodeToString(controlBlock),
	}
}

// UnwrapLeafScript unwraps the given leaf script
func UnwrapLeafScript(leafScript LeafScript) ([]byte, []byte, error) {
	script, err := hex.DecodeString(leafScript.Script)
	if err != nil {
		return nil, nil, err
	}

	controlBlock, err := hex.DecodeString(leafScript.ControlBlock)
	if err != nil {
		return nil, nil, err
	}

	return script, controlBlock, nil
}

// GetVaultUtxos gets the vault utxos from the given deposit txs
func GetVaultUtxos(depositTxs []*psbt.Packet, vaultPkScript []byte) ([]*btcbridgetypes.UTXO, error) {
	utxos := []*btcbridgetypes.UTXO{}

	for _, depositTx := range depositTxs {
		vaultUtxos, err := getVaultUtxosFromDepositTx(depositTx, vaultPkScript)
		if err != nil {
			return nil, err
		}

		utxos = append(utxos, vaultUtxos...)
	}

	return utxos, nil
}

// getVaultUtxosFromDepositTx gets vault utxos from the given deposit tx
func getVaultUtxosFromDepositTx(depositTx *psbt.Packet, vaultPkScript []byte) ([]*btcbridgetypes.UTXO, error) {
	utxos := []*btcbridgetypes.UTXO{}

	found := false

	for i, out := range depositTx.UnsignedTx.TxOut {
		if bytes.Equal(out.PkScript, vaultPkScript) {
			utxo := &btcbridgetypes.UTXO{
				Txid:         depositTx.UnsignedTx.TxHash().String(),
				Vout:         uint64(i),
				Amount:       uint64(out.Value),
				PubKeyScript: out.PkScript,
			}

			utxos = append(utxos, utxo)

			found = true
		}
	}

	if !found {
		return nil, ErrInvalidDepositTx
	}

	return utxos, nil
}

// getCetWitnessSize gets the cet witness size according to the given params
func getCetWitnessSize(cetType CetType, script []byte, controlBlock []byte) int {
	switch cetType {
	case CetType_LIQUIDATION, CetType_DEFAULT_LIQUIDATION:
		// dcm signature(64) + borrower signature(65) + len(script) + len(control block)
		return 64 + 65 + len(script) + len(controlBlock)

	default:
		// dcm signature(65) + borrower signature(65) + len(script) + len(control block)
		return 65 + 65 + len(script) + len(controlBlock)
	}
}
