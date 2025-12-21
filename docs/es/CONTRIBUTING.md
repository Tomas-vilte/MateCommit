# Cómo contribuir a MateCommit

¡Gracias por querer aportar a MateCommit! Soy yo, el único que está trabajando en este proyecto por ahora, pero si en algún momento alguien más se suma, estas son las reglas básicas para mantener todo en orden.

## Reglas generales

1. **Creación de tags**:
    - Los tags los manejo yo (el único admin por ahora jeje), nadie más puede crear, actualizar o borrar tags
    - Los tags se ponen solo después de que todo pase las verificaciones de estado, o sea, si las pruebas y el CI/CD no se quejan.

2. **Verificación del código**:
    - Antes de mandar un PR, asegurate de que tu código no esté todo roto. Si no pasa las pruebas, no voy a aceptar el PR xd, hasta que esté todo en ok.

3. **Cómo manejar los PRs**:
    - **Usá los templates**: Ya armé unos formularios bien pro en GitHub para reportar bugs o pedir nuevas funcionalidades. ¡Usalos! Me ayuda un montón a que no se me pase nada por alto.
    - Los PRs tienen que estar bien hechos. Asegurate de completar el **Template de Pull Request** que te aparece cuando abrís uno nuevo.
    - No seas vago, poné una descripción clara de qué hiciste y por qué lo hiciste. Nada de "cambié el código", explicá qué cosas cambiaron, para que el que vea sepa qué hiciste y tenga más contexto.

## Cómo hacer un Pull Request (PR)

Si querés sumar tu cambio, hacé lo siguiente:

1. **Forkea el repo**.
2. **Clonalo a tu máquina**.
3. **Creá una rama** con un nombre que se entienda qué vas a hacer (tipo `feature/agregar-nueva-funcionalidad` o `bugfix/corregir-error`).
4. **Hacés los cambios** y asegurate de que no estés rompiendo todo. ¡Testea bien, no seas pajero!
5. **Hacé un commit** con un mensaje claro. Usá este formato:
    - `tipo: descripción corta` (Ejemplo: `feat: agregar nueva funcionalidad` o `fix: corregir bug en validación`).
    - Si hace falta, explicá un toque más en el cuerpo del mensaje.
6. **Mandá el PR** desde tu rama hacia `master`.
    - Revisá que no haya conflictos y que el código esté bien.
    - Yo lo reviso, y si todo está ok, lo apruebo.

## Buenas prácticas

- **Mantené tu fork actualizado**: Antes de ponerte a cambiar cosas, asegurate de que tu fork esté sincronizado con el repo principal.
- **Escribí pruebas**: Si estás agregando algo nuevo, escribí pruebas para eso. Si corregís un bug, agregá una prueba para que no vuelva a pasar.
- **Documentá tus cambios**: Si hacés un cambio importante, ponélo en la documentación para que no dejemos a nadie perdido.

## Resumen

1. **Los tags son exclusivos de los admins**. Si no sos admin, no toques nada de eso.
2. **Tu código tiene que pasar las verificaciones antes de ser aprobado** Sino tabla.
3. **Seguí las buenas prácticas al hacer un PR**.

¡Gracias por querer sumar!

---
PD: perdón que sea tan rompe bolas con esto, pero aprendemos a usar buenas prácticas y aprender obviamente.
**MateCommit - Por ahora, soy yo. Si algún día alguien más se suma, les cuento las reglas también.**