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
