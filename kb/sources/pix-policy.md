# Politica de Transferencias PIX (2026)

Fonte: Politica de Pagamentos - PIX, revisao 2026-01. Os valores sao ilustrativos
para este desafio tecnico.

## 1. Limites diarios

Cada conta possui um limite diario de transferencias via PIX. O limite diario padrao
e de R$ 5.000,00 por conta. Esse limite pode ser configurado individualmente por
conta, conforme o perfil e o historico do cliente. Transferencias que ultrapassam o
limite diario disponivel sao rejeitadas pelo sistema.

## 2. Autenticacao adicional (step-up)

Transferencias de valor igual ou superior a R$ 1.000,00 exigem autenticacao adicional
do cliente, tambem chamada de step-up. Essa verificacao e independente da sessao ja
autenticada e tem como objetivo proteger operacoes de maior valor contra fraude.

A autenticacao adicional e realizada por meio de um codigo de uso unico (OTP) enviado
ao cliente. O codigo e vinculado a operacao especifica - valor e destinatario - de
modo que nao pode ser reaproveitado para uma operacao diferente.

## 3. Confirmacao explicita

Toda transferencia, independentemente do valor, exige confirmacao explicita do cliente,
que deve revisar o valor e o destinatario antes da execucao. A confirmacao e registrada
para fins de auditoria.

## 4. Registro e auditoria

Todas as transferencias, bem como tentativas rejeitadas por limite ou por falta de
autenticacao, sao registradas em trilha de auditoria com data, hora, usuario, valor e
resultado da operacao.
