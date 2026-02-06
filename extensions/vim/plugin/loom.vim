" loom.vim - Loom Integration for Vim/Neovim
" Maintainer: Loom Team
" Version: 1.0.0

if exists('g:loaded_loom')
  finish
endif
let g:loaded_loom = 1

" Configuration
let g:loom_api_endpoint = get(g:, 'loom_api_endpoint', 'http://localhost:8080')
let g:loom_api_key = get(g:, 'loom_api_key', '')
let g:loom_model = get(g:, 'loom_model', 'default')
let g:loom_enable_suggestions = get(g:, 'loom_enable_suggestions', 1)
let g:loom_max_context_lines = get(g:, 'loom_max_context_lines', 50)

" Commands
command! -nargs=? LoomChat call loom#chat#open(<q-args>)
command! -range LoomExplain call loom#actions#explain(<line1>, <line2>)
command! -range LoomGenerateTests call loom#actions#generate_tests(<line1>, <line2>)
command! -range LoomRefactor call loom#actions#refactor(<line1>, <line2>)
command! -range LoomFixBug call loom#actions#fix_bug(<line1>, <line2>)
command! LoomToggleSuggestions call loom#suggestions#toggle()

" Keymaps (optional, users can override)
if !exists('g:loom_no_default_keymaps')
  " Leader + a for Loom menu
  nnoremap <leader>ac :LoomChat<CR>
  vnoremap <leader>ae :LoomExplain<CR>
  vnoremap <leader>at :LoomGenerateTests<CR>
  vnoremap <leader>ar :LoomRefactor<CR>
  vnoremap <leader>af :LoomFixBug<CR>
  nnoremap <leader>as :LoomToggleSuggestions<CR>
endif

" Auto commands for inline suggestions
if g:loom_enable_suggestions && (has('nvim') || has('textprop'))
  augroup LoomSuggestions
    autocmd!
    autocmd InsertCharPre * call loom#suggestions#on_char()
    autocmd InsertLeave * call loom#suggestions#clear()
  augroup END
endif

" Health check (Neovim only)
if has('nvim')
  command! LoomHealth call loom#health#check()
endif
