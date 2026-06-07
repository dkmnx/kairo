# Taste (Continuously Learned by [CommandCode][cmd])

[cmd]: https://commandcode.ai/

# git
- Use short conventional commit messages (type(scope): description) — describe only what changed, no explanations or AI slop. Confidence: 0.75
- Do not add Co-authored-by or other attribution lines to commit messages. Confidence: 0.75

# audit
- Always run the /slop audit on changed code before committing, checking for AI-generated slop patterns (obvious comments, TODO placeholders, identity functions, robotic naming). Confidence: 0.85

# code-style
- Do not write AI-generated slop code — avoid obvious comments that restate the code, redundant doc strings, and architectural trivia in comments. Confidence: 0.85
- Always follow Google Go style conventions. Confidence: 0.85

# workflow
- Always run markdownlint-cli2 after modifying markdown files. Confidence: 0.90
