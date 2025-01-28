package main

// Lookup tables for decompiling assembly during debugging.

const (
	// 0 = NO PREFIX
	REX = 1 // REX prefix
	NIL = 2 // Null prefix
	TBI = 3 // Two byte instruction prefix
	OVR = 4 // Override prefix
	SSE = 5 // Streaming SIMD Extension
	LCK = 6 // Lock prefix
)

inst_prefixes := [256]int{
	//       00  01  02  03  04  05  06  07  08  09  0A  0B  0C  0D  0E  0F
	/* 00 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  TBI,
	/* 10 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* 20 */ 0,  0,  0,  0,  0,  0,  NIL,0,  0,  0,  0,  0,  0,  0,  NIL,0,
	/* 30 */ 0,  0,  0,  0,  0,  0,  NIL,0,  0,  0,  0,  0,  0,  0,  NIL,0,
	/* 40 */ REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,REX,
	/* 50 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* 60 */ 0,  0,  0,  0,  OVR,OVR,OVR,OVR,0,  0,  0,  0,  0,  0,  0,  0,
	/* 70 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* 80 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* 90 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* A0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* B0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* C0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* D0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* E0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* F0 */ LCK,0,SSE,SSE,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0
}

// One byte opcodes
// Prefixes and special instructions are marked as invalid

inst_opc_1b := [256]string{
	//       00  	01  	02  	03  	04  	05  	06  	07  	08  	09  	0A  	0B  	0C  	0D  	0E  	0F
	/* 00 */ "ADD", "ADD", 	"ADD",  "ADD",  "ADD",  "ADD",  "INV",  "INV",  "OR",  "OR",   "OR",   "OR",   "OR",   "OR",   "INV",  "INV",
	/* 10 */ "ADC", "ADC",  "ADC",  "ADC",  "ADC",  "ADC",  "INV",  "INV",  "SBB", "SBB",  "SBB",  "SBB",  "SBB",  "SBB",  "INV",  "INV",
	/* 20 */ "AND", "AND",  "AND",  "AND",  "AND",  "AND",  "INV",  "INV",  "SUB", "SUB",  "SUB",  "SUB",  "SUB",  "SUB",  "INV",  "INV",
	/* 30 */ "XOR", "XOR",  "XOR",  "XOR",  "XOR",  "XOR",  "INV",  "INV",  "CMP", "CMP",  "CMP",  "CMP",  "CMP",  "CMP",  "INV",  "INV",
	/* 40 */ "INV", "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV", "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV",
	/* 50 */ "PUSH","PUSH", "PUSH", "PUSH", "PUSH", "PUSH", "PUSH", "PUSH", "POP", "POP",  "POP",  "POP",  "POP",  "POP",  "POP",  "POP",
	/* 60 */ "INV", "INV",  "INV", "MOVXSD","INV",  "INV",  "INV",  "INV",  "PUSH","IMUL", "PUSH", "IMUL", "INSB", "INSW", "OUTSD","OUTSW",
	/* 70 */ "JO",  "JNO",  "JC",  "JNC",   "JZ",   "JNZ",  "JNA",  "JA",   "JS",  "JNS",  "JP",   "JNP",  "JL",   "JNL",  "JNG",  "JG",
	/* 80 */ "MOD", "MOD",  "INV", "MOD",   "TEST", "TEST", "XCHG", "XCHG", "MOV", "MOV",  "MOV",  "MOV",  "MOV",  "LEA",  "MOV",  "MOD", // FULL MOD R/M Byte 80-83, 81 invalid
	/* 90 */ "NOP", "XCHG", "XCHG","XCHG",  "XCHG", "XCHG", "XCHG", "XCHG", "CBW", "CWD",  "INV",  "WAIT", "PUSHF","POPF", "SAHF", "LAHF", // 90+r = XCHG with rAX, 90 = NOP, F3 prefix + 90 = PAUSE
	/* A0 */ "MOV", "MOV",  "MOV", "MOV",   "MOVS", "MOVS", "CMPS", "CMPS", "TEST","TEST", "STOS", "STOS", "LODS", "LODS", "SCAS", "SCAS",
	/* B0 */ "MOV", "MOV",  "MOV", "MOV",   "MOV",  "MOV",  "MOV",  "MOV",  "MOV", "MOV",  "MOV",  "MOV",  "MOV",  "MOV",  "MOV",  "MOV",
	/* C0 */ "MOD", "MOD",  "RET", "RET",   "INV",  "INV",  "MOD",  "MOD", "ENTER","LEAVE","RETF", "RETF", "INT",  "INT",  "INTO", "IRET",
	/* D0 */ "MOD", "MOD",  "MOD", "MOD",   "INV",  "INV",  "INV",  "XLAT", "MOD", "MOD",  "MOD",  "MOD",  "MOD",  "MOD",  "MOD",  "MOD",
	/* E0 */"LOOPNZ","LOOPZ","LOOP","JECXZ","IN",   "IN",   "OUT",  "OUT",  "CALL","JMP",  "INV",  "JMP",  "IN",   "IN",   "OUT",  "OUT",
	/* F0 */ "INV", "INT1", "INV", "INV",   "HLT",  "CMC",  "MOD",  "MOD",  "CLC", "STC",  "CLI",  "STI",  "CLD",  "STD",  "MOD",  "MOD"
}

// One byte opcodes with full modr/m byte

inst_modrm_1b := [192]string{
	//			0		1		2		3		4		5		6		7     // 8, 9 for easier indexing
	/* 80 */	"ADD",  "OR",   "ADC",  "SBB",  "AND",  "SUB",  "XOR",  "CMP", "", ""
	/* 81 */    "ADD",  "OR",   "ADC",  "SBB",  "AND",  "SUB",  "XOR",  "CMP", "", ""
	/* 83 */    "ADD",  "OR",   "ADC",  "SBB",  "AND",  "SUB",  "XOR",  "CMP", "", ""
	/* 8F */    "POP",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV", "", ""
	/* C0 */    "ROL",  "ROR",  "RCL",  "RCR",  "SHL",  "SHR",  "SAL",  "SAR", "", ""
	/* C1 */    "ROL",  "ROR",  "RCL",  "RCR",  "SHL",  "SHR",  "SAL",  "SAR", "", ""
	/* C6 */    "MOV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV", "", ""
	/* C7 */    "MOV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV",  "INV", "", ""
	/* D0 */
	/* D1 */
	/* D2 */
	/* D3 */
	/* D8 */
	/* D9 */
	/* DA */
	/* DB */
	/* DC */
	/* DD */
	/* DE */
	/* DF */
	/* F6 */
	/* F7 */
	/* FE */
	/* FF */

}

func Lookup(opcode int) string {

}
