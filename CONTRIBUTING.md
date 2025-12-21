# Contributing to MateCommit 🧉

Thanks for your interest in helping out with MateCommit! Right now, I'm the only one working on this project, but if you want to join in, here are some basic rules to keep things from getting messy.

## General Rules

1. **Tag Management**:
    - For now, I handle the tags exclusively. Please don't create, update, or delete tags on your own.
    - I only push tags once everything passes the status checks (tests, CI/CD, etc.).

2. **Code Verification**:
    - Before you send a Pull Request (PR), make sure your code isn't broken. If the tests don't pass, I won't be able to accept it until everything is green.

3. **How to handle PRs**:
    - **Use the Templates**: I've set up specific GitHub templates for Bug Reports and Feature Requests. Please use them! It helps me stay organized.
    - PRs need to be well-structured. Use the **Pull Request Template** that pops up when you open one.
    - Don't be vague; provide a clear description of what you did and why. Instead of just saying "updated code," explain the specific changes so I have more context when reviewing.

---

## How to make a Pull Request (PR)

If you've got a change you want to add, follow these steps:

1. **Fork the repo**.
2. **Clone it to your machine**.
3. **Create a branch** with a name that makes sense (like `feature/add-neat-thing` or `bugfix/fix-that-error`).
4. **Make your changes** and double-check you're not breaking everything. Test your stuff!
5. **Make a commit** with a clear message. I recommend this format:
    - `type: short description` (e.g., `feat: add awesome feature` or `fix: resolve validation bug`).
    - If needed, add some more details in the body of the message.
6. **Send the PR** from your branch to `master`.
    - Check for conflicts and ensure the code is clean.
    - I'll review it and, if everything looks good, I'll merge it.

---

## Best Practices

- **Keep your fork updated**: Before you start hacking away, make sure your fork is synced with the main repo.
- **Write tests**: If you're adding something new, write a test for it. If you've fixed a bug, add a test so it stays fixed.
- **Document your changes**: If you make a significant change, update the documentation so we don't leave anyone scratching their head.

## Quick Summary

1. **Tags are for admins only** (which is just me at the moment).
2. **Your code must pass all checks** before it gets approved.
3. **Follow the PR guidelines** for a smooth review.

Thanks for wanting to contribute!

---
*PD: Sorry if I'm being a bit strict with this, but it's the best way for us to learn and maintain high standards together.*
**MateCommit - For now, it's just me. If that changes, I'll let you know the new rules!**
