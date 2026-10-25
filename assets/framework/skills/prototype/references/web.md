# Web prototypes

Framework version: 0.3.7

- Prefer semantic HTML, CSS, vanilla JavaScript, and native browser APIs. Add a framework only when required
  interaction fidelity justifies its setup; keep styling plain unless visual exploration is requested.
- Implement navigation, forms, interactions, and state changes with a clear information hierarchy, consistent
  controls, and immediate feedback. Include relevant empty, loading, validation, success, and error states;
  provide deterministic ways to reach them.
- Use headings, links for navigation, buttons for actions, and associated form labels. Make interactions
  keyboard-operable with logical focus order, visible focus, and text identifying validation errors.
- Use `localStorage` for small simple persistent state; use IndexedDB when structured persistent browser data
  warrants it. Keep storage keys/databases prototype-specific so reset cannot clear unrelated origin data.
- Use a minimal local HTTP server when modules, fetch, or storage/origin behavior requires it; document one
  fixed local origin. Use `file://` only when the selected browser features work there reliably.
- Exercise the flows in a browser, including keyboard/form behavior, navigation, local assets, persistence,
  and console errors. If browser access is unavailable, report those checks as unverified.
