" loom/chat.vim - Chat interface

let s:chat_buffer = -1
let s:conversation_history = []

function! loom#chat#open(initial_message) abort
  " Create or focus chat buffer
  if s:chat_buffer == -1 || !bufexists(s:chat_buffer)
    call s:create_chat_buffer()
  else
    let l:win = bufwinnr(s:chat_buffer)
    if l:win == -1
      execute 'vsplit'
      execute 'buffer' s:chat_buffer
    else
      execute l:win . 'wincmd w'
    endif
  endif

  " Send initial message if provided
  if !empty(a:initial_message)
    call s:send_message(a:initial_message)
  endif
endfunction

function! s:create_chat_buffer() abort
  execute 'vsplit'
  execute 'enew'

  let s:chat_buffer = bufnr('%')
  setlocal buftype=nofile
  setlocal bufhidden=hide
  setlocal noswapfile
  setlocal filetype=loom-chat

  file Loom\ Chat

  " Add instructions
  call append(0, [
    \ '# Loom Chat',
    \ '',
    \ 'Type your message and press <CR> in insert mode to send.',
    \ 'Use :LoomChat to open this window.',
    \ '',
    \ '---',
    \ ''
  \ ])

  " Set up insert mode mapping
  inoremap <buffer> <CR> <Esc>:call <SID>send_current_line()<CR>
endfunction

function! s:send_current_line() abort
  let l:line = getline('.')

  if empty(trim(l:line))
    return
  endif

  call s:send_message(l:line)

  " Clear input line
  call setline('.', '')
endfunction

function! s:send_message(message) abort
  " Add user message to buffer
  call append('$', ['', 'You: ' . a:message, ''])

  " Add to conversation history
  call add(s:conversation_history, {'role': 'user', 'content': a:message})

  " Show loading
  call append('$', ['Loom: Thinking...', ''])
  redraw

  try
    " Get response
    let l:response = loom#client#send_message(s:conversation_history)

    " Remove loading message
    call deletebufline(s:chat_buffer, '$')
    call deletebufline(s:chat_buffer, '$')

    " Add assistant response
    call append('$', ['Loom: ' . l:response, ''])

    " Add to history
    call add(s:conversation_history, {'role': 'assistant', 'content': l:response})

    " Scroll to bottom
    normal! G
  catch
    " Remove loading message
    call deletebufline(s:chat_buffer, '$')
    call deletebufline(s:chat_buffer, '$')

    " Show error
    call append('$', ['Error: ' . v:exception, ''])
    echoerr v:exception
  endtry
endfunction

function! loom#chat#clear() abort
  let s:conversation_history = []
  if bufexists(s:chat_buffer)
    call deletebufline(s:chat_buffer, 1, '$')
    call s:create_chat_buffer()
  endif
endfunction
