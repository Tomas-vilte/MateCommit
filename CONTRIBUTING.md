# Cómo contribuir a MateCommit

¡Gracias por querer aportar a MateCommit! Soy yo, el único que está trabajando en este proyecto por ahora, pero si en algún momento alguien más se suma, estas son las reglas básicas para mantener todo en orden.

## Reglas generales

1. **Creación de tags**:
   - Los tags los manejo yo (el único admin por ahora jeje), nadie más puede crear, actualizar o borrar tags
   - Los tags se ponen solo después de que todo pase las verificaciones de estado, o sea, si las pruebas y el CI/CD no se quejan.

2. **Verificación del código**:
   - Antes de mandar un PR, asegurate de que tu código no esté todo roto. Si no pasa las pruebas, olvidate, no vamos a aceptar el PR hasta que esté todo en ok.

3. **Cómo manejar los PRs**:
   - Los PRs tienen que estar bien hechos.
   - No seas vago, pone una descripción clara de qué hiciste y por qué lo hiciste. Nada de "cambié el código", explicá qué cambiaron.

## Cómo hacer un Pull Request (PR)

Si querés sumar tu cambio, hacé lo siguiente:

1. **Forkea el repo**.
2. **Clonalo a tu máquina**.
3. **Crea una rama** con un nombre que se entienda qué vas a hacer (tipo `feature/agregar-nueva-funcionalidad` o `bugfix/corregir-error`).
4. **Hacés los cambios** y asegurate de que no estés rompiendo todo. ¡Testea bien, no seas pajero!
5. **Hace un commit** con un mensaje claro. Usá este formato:
   - `tipo: descripción corta` (Ejemplo: `feat: agregar nueva funcionalidad` o `fix: corregir bug en validación`).
   - Si hace falta, explicá un toque más en el cuerpo del mensaje.
6. **Mandá el PR** desde tu rama hacia `master`.
   - Revisá que no haya conflictos y que el código esté bien.
   - Yo lo reviso, y si todo está ok, lo apruebo.

## Política de Tags

1. **Los tags son exclusivos de los admins**. Si no sos admin, olvidate, no vas a poder tocar nada de eso.
2. Los tags son para marcar versiones estables, así que no te pongas a poner tags por cualquier pavada.
3. Los tags solo se crean después de que todo haya pasado las verificaciones de estado (pruebas, CI/CD, lo que sea).

## Buenas prácticas

- **Mantené tu fork actualizado**: Antes de ponerte a cambiar cosas, asegurate de que tu fork esté sincronizado con el repo principal, si no vas a tener un quilombo después.
- **Escribí pruebas**: Si estás agregando algo nuevo, escribí pruebas para eso. Si corregís un bug, agregá una prueba para que no vuelva a pasar.
- **Documentá tus cambios**: Si haces un cambio importante, ponélo en la documentación para que no dejemos a nadie perdido.

## Resumen

1. **Los tags son exclusivos de los admins**. Si no sos admin, no toques nada de eso.
2. **Tu código tiene que pasar las verificaciones antes de ser aprobado**. No te hagas el vivo.
3. **Seguí las buenas prácticas al hacer un PR**.

¡Gracias por querer sumar!

---

**MateCommit - Por ahora, soy yo. Si algún día alguien más se suma, les cuento las reglas también.**
