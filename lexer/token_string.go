// Code generated by "stringer -type=Token"; DO NOT EDIT.

package lexer

import "strconv"

const _Token_name = "NILEOFERRORIDENTIFIERNUMBERSTRINGWHITESPACECOMMENTNEWLINEHASHCOMMACOLONSEMICOLONATL_PARENR_PARENR_BRACKETL_CURLYR_CURLYRANGETORANGELENNOTBITNOTNEWL_BRACKETDOTQ_MARKISISNTMATCHMATCHNOTLTLTEGTGTEASSOC_STARTANDORBITORBITANDBITXORADDSUBCATMULDIVASSOC_ENDMODLSHIFTRSHIFTINCDECASSIGN_STARTEQADDEQSUBEQCATEQMULEQDIVEQMODEQLSHIFTEQRSHIFTEQBITOREQBITANDEQBITXOREQASSIGN_ENDINBOOLBREAKBUFFERCALLBACKCASECATCHCHARCLASSCONTINUECREATEDEFAULTDLLDODOUBLEELSEFALSEFLOATFORFOREVERFUNCTIONGDIOBJHANDLEIFINT64LONGRESOURCERETURNSHORTSTRUCTSWITCHSUPERTHISTHROWTRUETRYVOIDWHILEALTERAVERAGECASCADECOUNTDELETEDROPENSUREEXTENDHISTORYINDEXINSERTINTERSECTINTOJOINKEYLEFTJOINLISTMAXMINMINUSPROJECTREMOVERENAMEREVERSESETSORTSUMMARIZESVIEWTIMESTOTOTALUNIONUNIQUEUPDATEUPDATESVIEWWHERE"

var _Token_index = [...]uint16{0, 3, 6, 11, 21, 27, 33, 43, 50, 57, 61, 66, 71, 80, 82, 89, 96, 105, 112, 119, 126, 134, 137, 143, 146, 155, 158, 164, 166, 170, 175, 183, 185, 188, 190, 193, 204, 207, 209, 214, 220, 226, 229, 232, 235, 238, 241, 250, 253, 259, 265, 268, 271, 283, 285, 290, 295, 300, 305, 310, 315, 323, 331, 338, 346, 354, 364, 366, 370, 375, 381, 389, 393, 398, 402, 407, 415, 421, 428, 431, 433, 439, 443, 448, 453, 456, 463, 471, 477, 483, 485, 490, 494, 502, 508, 513, 519, 525, 530, 534, 539, 543, 546, 550, 555, 560, 567, 574, 579, 585, 589, 595, 601, 608, 613, 619, 628, 632, 636, 639, 647, 651, 654, 657, 662, 669, 675, 681, 688, 691, 695, 704, 709, 714, 716, 721, 726, 732, 738, 745, 749, 754}

func (i Token) String() string {
	if i >= Token(len(_Token_index)-1) {
		return "Token(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Token_name[_Token_index[i]:_Token_index[i+1]]
}