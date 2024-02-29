class Solution:
    def wordPattern(self, pattern: str, s: str) -> bool:
        s = s.split(" ")
        if len(pattern) != len(s) or len(set(pattern)) != len(set(s)):
            return False
        
        d = dict()
        for i in range(len(s)):
            if s[i] not in d:
                d[pattern[i]] = s[i]
            elif s[i] != d[pattern[i]]:
                return False
        return True
    
