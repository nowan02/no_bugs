package main

// Lookup tables for decompiling assembly during debugging.

const (
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
// Prefixes are marked as invalid

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
	/* 80 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* 90 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* A0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* B0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* C0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* D0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* E0 */ 0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,
	/* F0 */ LCK,0,SSE,SSE,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0,  0
}

func Lookup(opcode int) string {

}
