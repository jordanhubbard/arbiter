" loom/client.vim - API client

function! loom#client#send_message(messages, ...) abort
  let l:model = get(a:, 1, g:loom_model)

  let l:request = {
    \ 'messages': a:messages,
    \ 'model': l:model,
    \ 'temperature': 0.7,
    \ 'max_tokens': 2000
  \ }

  let l:url = g:loom_api_endpoint . '/api/v1/chat/completions'
  let l:headers = ['Content-Type: application/json']

  if !empty(g:loom_api_key)
    call add(l:headers, 'Authorization: Bearer ' . g:loom_api_key)
  endif

  let l:response = loom#client#http_post(l:url, json_encode(l:request), l:headers)

  if l:response.status == 200
    let l:data = json_decode(l:response.body)
    if has_key(l:data, 'choices') && len(l:data.choices) > 0
      return l:data.choices[0].message.content
    endif
  endif

  throw 'Loom API error: ' . l:response.status . ' - ' . l:response.body
endfunction

function! loom#client#health_check() abort
  let l:url = g:loom_api_endpoint . '/health'
  let l:response = loom#client#http_get(l:url, [])
  return l:response.status == 200
endfunction

function! loom#client#http_post(url, data, headers) abort
  if has('nvim')
    return s:http_request_nvim('POST', a:url, a:data, a:headers)
  else
    return s:http_request_vim('POST', a:url, a:data, a:headers)
  endif
endfunction

function! loom#client#http_get(url, headers) abort
  if has('nvim')
    return s:http_request_nvim('GET', a:url, '', a:headers)
  else
    return s:http_request_vim('GET', a:url, '', a:headers)
  endif
endfunction

" Neovim implementation using curl via job
function! s:http_request_nvim(method, url, data, headers) abort
  let l:curl_cmd = ['curl', '-s', '-X', a:method, '-w', '\n%{http_code}']

  for l:header in a:headers
    call extend(l:curl_cmd, ['-H', l:header])
  endfor

  if !empty(a:data)
    call extend(l:curl_cmd, ['-d', a:data])
  endif

  call add(l:curl_cmd, a:url)

  let l:output = system(join(l:curl_cmd, ' '))
  let l:lines = split(l:output, '\n')
  let l:status = str2nr(l:lines[-1])
  let l:body = join(l:lines[:-2], '\n')

  return {'status': l:status, 'body': l:body}
endfunction

" Vim 8 implementation
function! s:http_request_vim(method, url, data, headers) abort
  " Same as Neovim for now - use system curl
  return s:http_request_nvim(a:method, a:url, a:data, a:headers)
endfunction
