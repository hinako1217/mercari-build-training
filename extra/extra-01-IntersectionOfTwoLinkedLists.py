# Definition for singly-linked list.
# class ListNode:
#     def __init__(self, x):
#         self.val = x
#         self.next = None

class Solution:
    def getIntersectionNode(self, headA: ListNode, headB: ListNode) -> Optional[ListNode]:
        countA = 0
        countB = 0
        tmpA = headA
        tmpB = headB

        while tmpA:
            countA += 1
            tmpA = tmpA.next
        while tmpB:
            countB += 1
            tmpB = tmpB.next

        tmpA = headA
        tmpB = headB

        if countA > countB:
            for i in range(countA-countB):
                tmpA = tmpA.next
        else:
            for i in range(countB-countA):
                tmpB = tmpB.next
        
        while tmpA:
            if tmpA == tmpB:
                return tmpA
            tmpA = tmpA.next
            tmpB = tmpB.next
        
        return None
