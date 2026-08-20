hs.keycodes.inputSourceChanged(function(v)
    local inputSource = {
      english = "com.apple.keylayout.ABC",
      korean = "com.apple.inputmethod.Korean.2SetKorean",
    }
  
    local current = hs.keycodes.currentSourceID()
    local language = nil
  
    if current == inputSource.korean then
      language = '한글'
    elseif current == inputSource.english then
      language = 'English'
    else
      language = current
    end
  
    hs.alert.closeAll()
    hs.alert.show(language, {
        atScreenEdge = 2,
        textSize = 16,
        padding = 10,
        radius = 8,
    } ,0.5)
  end)