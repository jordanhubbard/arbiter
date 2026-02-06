# Loom Vim/Neovim Plugin

AI-powered coding assistant for Vim and Neovim.

## Features

- **Chat Interface**: Split window chat with conversation history
- **Code Actions**: Commands for code assistance
- **Inline Suggestions**: AI-powered code completions (Neovim only)
- **Visual Mode**: Work with selected code
- **Keyboard Shortcuts**: Efficient workflows

## Installation

### Using vim-plug

```vim
Plug 'jordanhubbard/Loom', {'rtp': 'extensions/vim'}
```

### Using Vundle

```vim
Plugin 'jordanhubbard/Loom', {'rtp': 'extensions/vim'}
```

### Using Pathogen

```bash
cd ~/.vim/bundle
git clone https://github.com/jordanhubbard/Loom.git
ln -s Loom/extensions/vim loom
```

### Manual

```bash
cp -r extensions/vim/* ~/.vim/
# or for Neovim:
cp -r extensions/vim/* ~/.config/nvim/
```

## Configuration

Add to `.vimrc` or `init.vim`:

```vim
" Loom Configuration
let g:loom_api_endpoint = 'http://localhost:8080'
let g:loom_api_key = ''  " Optional
let g:loom_model = 'default'
let g:loom_enable_suggestions = 1
let g:loom_max_context_lines = 50

" Disable default keymaps (optional)
let g:loom_no_default_keymaps = 1
```

## Usage

### Commands

```vim
:LoomChat               " Open chat window
:LoomChat Hello         " Open chat with message

" Visual mode commands (select code first)
:'<,'>LoomExplain       " Explain selected code
:'<,'>LoomGenerateTests " Generate tests
:'<,'>LoomRefactor      " Refactor suggestions
:'<,'>LoomFixBug        " Debug help

" Toggle inline suggestions
:LoomToggleSuggestions
```

### Default Keymaps

```vim
<leader>ac  " Open chat
<leader>ae  " Explain selection (visual mode)
<leader>at  " Generate tests (visual mode)
<leader>ar  " Refactor (visual mode)
<leader>af  " Fix bug (visual mode)
<leader>as  " Toggle suggestions
```

### Chat Interface

1. Run `:LoomChat`
2. Type message
3. Press `<CR>` in insert mode to send
4. View response in same buffer

### Code Actions

1. Select code (visual mode)
2. Run command (e.g., `:'<,'>LoomExplain`)
3. View result in chat buffer

### Inline Suggestions (Neovim only)

- Start typing code
- Suggestions appear as virtual text
- Press `<Tab>` to accept
- Press `<Esc>` to dismiss

## Requirements

- Vim 8.0+ or Neovim 0.5+
- `curl` command line tool
- Loom server running

## Troubleshooting

### Connection Error

```vim
:echo loom#client#health_check()
```

Should return `1` if server is reachable.

### Check Configuration

```vim
:echo g:loom_api_endpoint
:echo g:loom_enable_suggestions
```

### Debug Mode

```vim
:set verbose=9
:LoomChat Test
```

## Advanced Configuration

### Custom Keymaps

```vim
let g:loom_no_default_keymaps = 1

nmap <C-a> :LoomChat<CR>
vmap <C-e> :LoomExplain<CR>
vmap <C-t> :LoomGenerateTests<CR>
```

### Model Selection

```vim
let g:loom_model = 'gpt-4'
```

### API Key from Environment

```vim
let g:loom_api_key = $LOOM_API_KEY
```

## Architecture

```
plugin/loom.vim       - Plugin initialization
autoload/loom/
  client.vim                - API client
  chat.vim                  - Chat interface
  actions.vim               - Code actions
  suggestions.vim           - Inline completions
  health.vim                - Health checks
```

## Development

### Testing

```bash
# Vim
vim -u test/vimrc test/test_chat.vim

# Neovim
nvim -u test/vimrc test/test_chat.vim
```

### Debugging

```vim
:set verbose=9
:messages
```

## Contributing

1. Fork repository
2. Create feature branch
3. Test with Vim and Neovim
4. Submit pull request

## License

MIT

---

**Powered by Loom** 🚀
