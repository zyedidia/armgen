A small tool for analyzing Arm's [Machine Readable Specification](https://developer.arm.com/downloads/-/exploration-tools).

For example, it can tell you all the instructions that branch in the base ISA:

```
$ armgen -branch ./ISA_A64_xml_A_profile-2023-06
b_cond.xml: [B]
b_uncond.xml: [B]
bl.xml: [BL]
blr.xml: [BLR]
br.xml: [BR]
cbnz.xml: [CBNZ]
cbz.xml: [CBZ]
ret.xml: [RET]
tbnz.xml: [TBNZ]
tbz.xml: [TBZ]
```

Or all instructions that branch, including all extensions to the ARMv8 ISA:

```
$ armgen -base=false -branch ./ISA_A64_xml_A_profile-2023-06
b_cond.xml: [B]
b_uncond.xml: [B]
bc_cond.xml: [BC]
bl.xml: [BL]
blr.xml: [BLR]
blra.xml: [BLRABZ BLRAB BLRAAZ BLRAA]
br.xml: [BR]
bra.xml: [BRAAZ BRAA BRABZ BRAB]
cbnz.xml: [CBNZ]
cbz.xml: [CBZ]
ret.xml: [RET]
reta.xml: [RETAA RETAB]
tbnz.xml: [TBNZ]
tbz.xml: [TBZ]
```

It can also display instruction encodings:

```
$ armgen -branch ./ISA_A64_xml_A_profile-2023-06
b_cond.xml: [B]
        iclass_br19: 0101010|o1=0|imm19=xxxxxxxxxxxxxxxxxxx|o0=0|cond=xxxx
b_uncond.xml: [B]
        iclass_br26: op=0|00101|imm26=xxxxxxxxxxxxxxxxxxxxxxxxxx
bl.xml: [BL]
        iclass_br26: op=1|00101|imm26=xxxxxxxxxxxxxxxxxxxxxxxxxx
blr.xml: [BLR]
        iclass_general: 1101011|Z=0|opc[2:1]=0|op=01|op2=11111|op3[5:2]=0000|A=0|M=0|Rn=xxxxx|Rm=00000
br.xml: [BR]
        iclass_general: 1101011|Z=0|opc[2:1]=0|op=00|op2=11111|op3[5:2]=0000|A=0|M=0|Rn=xxxxx|Rm=00000
cbnz.xml: [CBNZ]
        iclass_br19: sf=x|011010|op=1|imm19=xxxxxxxxxxxxxxxxxxx|Rt=xxxxx
cbz.xml: [CBZ]
        iclass_br19: sf=x|011010|op=0|imm19=xxxxxxxxxxxxxxxxxxx|Rt=xxxxx
ret.xml: [RET]
        iclass_general: 1101011|Z=0|opc[2:1]=0|op=10|op2=11111|op3[5:2]=0000|A=0|M=0|Rn=xxxxx|Rm=00000
tbnz.xml: [TBNZ]
        iclass_br14: b5=x|011011|op=1|b40=xxxxx|imm14=xxxxxxxxxxxxxx|Rt=xxxxx
tbz.xml: [TBZ]
        iclass_br14: b5=x|011011|op=0|b40=xxxxx|imm14=xxxxxxxxxxxxxx|Rt=xxxxx
```
